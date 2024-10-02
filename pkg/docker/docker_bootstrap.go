package docker

import (
	"context"
	"fmt"
	"kdeps/pkg/environment"
	"kdeps/pkg/logging"
	"kdeps/pkg/workflow"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// BootstrapDockerSystem initializes the Docker system and pulls models after ollama server is ready
func BootstrapDockerSystem(fs afero.Fs, ctx context.Context, environ *environment.Environment) (bool, error) {
	var apiServerMode bool

	if environ.DockerMode == "1" {
		logging.Info("Inside Docker environment. Proceeding with bootstrap.")
		logging.Info("Initializing Docker system")

		agentDir := "/agent"
		apiServerPath := filepath.Join(agentDir, "/actions/api")
		agentWorkflow := filepath.Join(agentDir, "workflow/workflow.pkl")
		wfCfg, err := workflow.LoadWorkflow(ctx, agentWorkflow)
		if err != nil {
			logging.Error("Error loading workflow: ", err)
			return apiServerMode, err
		}

		// Parse OLLAMA_HOST to get the host and port
		host, port, err := parseOLLAMAHost()
		if err != nil {
			return apiServerMode, err
		}

		// Start ollama server in the background
		if err := startOllamaServer(); err != nil {
			return apiServerMode, fmt.Errorf("Failed to start ollama server: %v", err)
		}

		// Wait for ollama server to be fully ready (using the parsed host and port)
		err = waitForServer(host, port, 60*time.Second)
		if err != nil {
			return apiServerMode, err
		}

		// Once ollama server is ready, proceed with pulling models
		wfSettings := *wfCfg.Settings
		apiServerMode = wfSettings.ApiServerMode

		dockerSettings := *wfSettings.AgentSettings
		modelList := dockerSettings.Models
		for _, value := range modelList {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			logging.Info("Pulling model: ", value)
			stdout, stderr, exitCode, err := KdepsExec("ollama", []string{"pull", value})
			if err != nil {
				logging.Error("Error pulling model: ", value, " stdout: ", stdout, " stderr: ", stderr, " exitCode: ", exitCode, " err: ", err)
				return apiServerMode, fmt.Errorf("Error pulling model %s: %s %s %d %v", value, stdout, stderr, exitCode, err)
			}
		}

		if err := fs.MkdirAll(apiServerPath, 0777); err != nil {
			return true, err
		}

		go func() error {
			if err := StartApiServerMode(fs, ctx, wfCfg, environ, apiServerPath); err != nil {
				return err
			}

			return nil
		}()
	}

	logging.Info("Docker system bootstrap completed.")

	return apiServerMode, nil
}

func CreateFlagFile(fs afero.Fs, filename string) error {
	// Check if file exists
	if exists, err := afero.Exists(fs, filename); err != nil {
		return err
	} else if !exists {
		// Create the file if it doesn't exist
		file, err := fs.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	} else {
		// If the file exists, update its modification time to the current time
		currentTime := time.Now().Local()
		if err := fs.Chtimes(filename, currentTime, currentTime); err != nil {
			return err
		}
	}
	return nil
}
