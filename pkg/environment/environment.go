package environment

import (
	"context"
	"os"
	"path/filepath"

	env "github.com/Netflix/go-env"
	"github.com/spf13/afero"
)

const SystemConfigFileName = ".kdeps.pkl"

// Environment holds environment configurations loaded from the OS or defaults.
type Environment struct {
	Root           string `env:"ROOT_DIR,default=/"`
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	KdepsConfig    string `env:"KDEPS_CONFIG,default=$HOME/.kdeps.pkl"`
	DockerMode     string `env:"DOCKER_MODE,default=0"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

// checkConfig checks if the .kdeps.pkl file exists in the given directory.
func checkConfig(fs afero.Fs, ctx context.Context, baseDir string) (string, error) {
	configFile := filepath.Join(baseDir, SystemConfigFileName)
	if exists, err := afero.Exists(fs, configFile); err == nil && exists {
		return configFile, nil
	} else {
		return "", err
	}
	return "", nil
}

// findKdepsConfig searches for the .kdeps.pkl file in both the Pwd and Home directories.
func findKdepsConfig(fs afero.Fs, ctx context.Context, pwd, home string) string {
	// Check for kdeps config in Pwd directory
	if configFile, _ := checkConfig(fs, ctx, pwd); configFile != "" {
		return configFile
	}
	// Check for kdeps config in Home directory
	if configFile, _ := checkConfig(fs, ctx, home); configFile != "" {
		return configFile
	}
	return ""
}

// isDockerEnvironment checks for the presence of Docker-related indicators.
func isDockerEnvironment(fs afero.Fs, ctx context.Context, root string) bool {
	dockerEnvFlag := filepath.Join(root, ".dockerenv")
	if exists, _ := afero.Exists(fs, dockerEnvFlag); exists {
		// Ensure all required Docker environment variables are set
		return allDockerEnvVarsSet(ctx)
	}
	return false
}

// allDockerEnvVarsSet checks if required Docker environment variables are set.
func allDockerEnvVarsSet(ctx context.Context) bool {
	requiredVars := []string{"SCHEMA_VERSION", "OLLAMA_HOST", "KDEPS_HOST"}
	for _, v := range requiredVars {
		if value, exists := os.LookupEnv(v); !exists || value == "" {
			return false
		}
	}
	return true
}

// NewEnvironment initializes and returns a new Environment based on provided or default settings.
func NewEnvironment(fs afero.Fs, ctx context.Context, environ *Environment) (*Environment, error) {
	if environ != nil {
		// If an environment is provided, prioritize overriding configurations
		kdepsConfigFile := findKdepsConfig(fs, ctx, environ.Pwd, environ.Home)
		dockerMode := "0"
		if isDockerEnvironment(fs, ctx, environ.Root) {
			dockerMode = "1"
		}

		return &Environment{
			Root:           environ.Root,
			Home:           environ.Home,
			Pwd:            environ.Pwd,
			KdepsConfig:    kdepsConfigFile,
			NonInteractive: "1", // Prioritize non-interactive mode for overridden environments
			DockerMode:     dockerMode,
		}, nil
	}

	// Load environment variables into a new Environment struct
	environment := &Environment{}
	extras, err := env.UnmarshalFromEnviron(environment)
	if err != nil {
		return nil, err
	}
	environment.Extras = extras

	// Find kdepsConfig file and check if running in Docker
	kdepsConfigFile := findKdepsConfig(fs, ctx, environment.Pwd, environment.Home)
	dockerMode := "0"
	if isDockerEnvironment(fs, ctx, environment.Root) {
		dockerMode = "1"
	}

	return &Environment{
		Root:        environment.Root,
		Home:        environment.Home,
		Pwd:         environment.Pwd,
		KdepsConfig: kdepsConfigFile,
		DockerMode:  dockerMode,
		Extras:      environment.Extras,
	}, nil
}
