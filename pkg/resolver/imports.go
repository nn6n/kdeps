package resolver

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kdeps/pkg/data"
	"github.com/kdeps/kdeps/pkg/schema"

	pklData "github.com/kdeps/schema/gen/data"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHttp "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/kdeps/schema/gen/utils"
	"github.com/spf13/afero"
)

func (dr *DependencyResolver) PrependDynamicImports(pklFile string) error {
	// Read the file content
	content, err := afero.ReadFile(dr.Fs, pklFile)
	if err != nil {
		return err
	}
	contentStr := string(content)

	// Define a regular expression to match "{{value}}"
	re := regexp.MustCompile(`\@\((.*)\)`)

	// Define ImportConfig struct
	type ImportConfig struct {
		Alias string
		Check bool // Flag to specify if the file existence should be checked
	}

	// Import configurations
	importCheck := map[string]ImportConfig{
		"pkl:json":     {Alias: "", Check: false},
		"pkl:test":     {Alias: "", Check: false},
		"pkl:math":     {Alias: "", Check: false},
		"pkl:platform": {Alias: "", Check: false},
		"pkl:semver":   {Alias: "", Check: false},
		"pkl:shell":    {Alias: "", Check: false},
		"pkl:xml":      {Alias: "", Check: false},
		"pkl:yaml":     {Alias: "", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Document.pkl", schema.SchemaVersion()): {Alias: "document", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Skip.pkl", schema.SchemaVersion()):     {Alias: "skip", Check: false},
		fmt.Sprintf("package://schema.kdeps.com/core@%s#/Utils.pkl", schema.SchemaVersion()):    {Alias: "utils", Check: false},
		filepath.Join(dr.ActionDir, "/llm/"+dr.RequestId+"__llm_output.pkl"):                    {Alias: "llm", Check: true},
		filepath.Join(dr.ActionDir, "/client/"+dr.RequestId+"__client_output.pkl"):              {Alias: "client", Check: true},
		filepath.Join(dr.ActionDir, "/exec/"+dr.RequestId+"__exec_output.pkl"):                  {Alias: "exec", Check: true},
		filepath.Join(dr.ActionDir, "/python/"+dr.RequestId+"__python_output.pkl"):              {Alias: "python", Check: true},
		filepath.Join(dr.ActionDir, "/data/"+dr.RequestId+"__data_output.pkl"):                  {Alias: "data", Check: true},
		dr.RequestPklFile: {Alias: "request", Check: true},
	}

	// Helper to check file existence
	fileExists := func(file string) bool {
		exists, _ := afero.Exists(dr.Fs, file)
		return exists
	}

	// Helper to generate import lines
	generateImportLine := func(file, alias string) string {
		if alias == "" {
			return fmt.Sprintf(`import "%s"`, file)
		}
		return fmt.Sprintf(`import "%s" as %s`, file, alias)
	}

	// Construct the dynamic import lines
	var importBuilder strings.Builder
	for file, config := range importCheck {
		if config.Check && !fileExists(file) {
			continue
		}

		importLine := generateImportLine(file, config.Alias)
		if !strings.Contains(contentStr, importLine) {
			importBuilder.WriteString(importLine + "\n")
		}
	}

	// If there are no new imports, return early
	importFiles := importBuilder.String()
	if importFiles == "" {
		return nil
	}

	// Add the imports after the "amends" line
	amendsIndex := strings.Index(contentStr, "amends")
	if amendsIndex != -1 {
		amendsLineEnd := strings.Index(contentStr[amendsIndex:], "\n") + amendsIndex + 1
		newContent := contentStr[:amendsLineEnd] + importFiles + contentStr[amendsLineEnd:]
		newContent = re.ReplaceAllString(newContent, `\($1)`)

		// Write the updated content back to the file
		err = afero.WriteFile(dr.Fs, pklFile, []byte(newContent), 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareImportFiles() error {
	files := map[string]string{
		"llm":    filepath.Join(dr.ActionDir, "/llm/"+dr.RequestId+"__llm_output.pkl"),
		"client": filepath.Join(dr.ActionDir, "/client/"+dr.RequestId+"__client_output.pkl"),
		"exec":   filepath.Join(dr.ActionDir, "/exec/"+dr.RequestId+"__exec_output.pkl"),
		"python": filepath.Join(dr.ActionDir, "/python/"+dr.RequestId+"__python_output.pkl"),
		"data":   filepath.Join(dr.ActionDir, "/data/"+dr.RequestId+"__data_output.pkl"),
	}

	for key, file := range files {
		dir := filepath.Dir(file)
		if err := dr.Fs.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", key, err)
		}

		// Check if the file exists, if not, create it
		exists, err := afero.Exists(dr.Fs, file)
		if err != nil {
			return fmt.Errorf("failed to check if %s file exists: %w", key, err)
		}

		if !exists {
			// Create the file if it doesn't exist
			f, err := dr.Fs.Create(file)
			if err != nil {
				return fmt.Errorf("failed to create %s file: %w", key, err)
			}
			defer f.Close()

			// Use packageUrl in the header writing
			packageUrl := fmt.Sprintf("package://schema.kdeps.com/core@%s#/", schema.SchemaVersion())
			writer := bufio.NewWriter(f)

			var schemaFile, blockType string
			switch key {
			case "exec":
				schemaFile = "Exec.pkl"
				blockType = "resources"
			case "python":
				schemaFile = "Python.pkl"
				blockType = "resources"
			case "client":
				schemaFile = "Http.pkl"
				blockType = "resources"
			case "llm":
				schemaFile = "LLM.pkl"
				blockType = "resources"
			case "data":
				schemaFile = "Data.pkl"
				blockType = "files" // Special case for "data"
			}

			// Write header using packageUrl and schemaFile
			if _, err := writer.WriteString(fmt.Sprintf("extends \"%s%s\"\n\n", packageUrl, schemaFile)); err != nil {
				return fmt.Errorf("failed to write header for %s: %w", key, err)
			}

			// Write the block (resources or files)
			if _, err := writer.WriteString(fmt.Sprintf("%s {\n}\n", blockType)); err != nil {
				return fmt.Errorf("failed to write block for %s: %w", key, err)
			}

			// Flush the writer
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("failed to flush output for %s: %w", key, err)
			}
		}
	}

	return nil
}

func (dr *DependencyResolver) PrepareWorkflowDir() error {
	src := dr.ProjectDir
	dest := dr.AgentDir
	fs := dr.Fs

	// Check if the destination exists and remove it if it does
	exists, err := afero.Exists(fs, dest)
	if err != nil {
		return fmt.Errorf("failed to check if destination exists: %w", err)
	}
	if exists {
		if err := fs.RemoveAll(dest); err != nil {
			return fmt.Errorf("failed to remove existing destination: %w", err)
		}
	}

	// Walk through the source directory
	err = afero.Walk(fs, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Determine the relative path and destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			// Create directories in the destination
			if err := fs.MkdirAll(targetPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		} else {
			// Copy file contents to the destination
			in, err := fs.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()

			out, err := fs.Create(targetPath)
			if err != nil {
				return err
			}
			defer out.Close()

			// Copy file contents
			if _, err := io.Copy(out, in); err != nil {
				return err
			}

			// Set file permissions to match the source file
			if err := fs.Chmod(targetPath, info.Mode()); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (dr *DependencyResolver) AddPlaceholderImports(filePath string) error {
	// Open the file using afero file system (dr.Fs)
	file, err := dr.Fs.Open(filePath)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Use a regular expression to find the id in the file
	re := regexp.MustCompile(`id\s*=\s*"([^"]+)"`)
	var actionId string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if the line contains the id
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			actionId = matches[1]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	if actionId == "" {
		return fmt.Errorf("action id not found in the file")
	}

	// Create placeholder entries using the parsed actionId
	type DataImpl struct {
		*utils.UtilsImpl

		// Files in the data folder mapped with the agent name and version
		Files *map[string]map[string]string `pkl:"files"`
	}

	dataFileList, err := data.PopulateDataFileRegistry(dr.Fs, dr.DataDir)
	dataFiles := &pklData.DataImpl{
		Files: dataFileList,
	}
	llmChat := &pklLLM.ResourceChat{}
	execCmd := &pklExec.ResourceExec{}
	pythonCmd := &pklPython.ResourcePython{}
	httpClient := &pklHttp.ResourceHTTPClient{
		Method: "GET",
	}

	if err := dr.AppendDataEntry(actionId, dataFiles); err != nil {
		return err
	}

	if err := dr.AppendChatEntry(actionId, llmChat); err != nil {
		return err
	}

	if err := dr.AppendExecEntry(actionId, execCmd); err != nil {
		return err
	}

	if err := dr.AppendHttpEntry(actionId, httpClient); err != nil {
		return err
	}

	if err := dr.AppendPythonEntry(actionId, pythonCmd); err != nil {
		return err
	}

	return nil
}
