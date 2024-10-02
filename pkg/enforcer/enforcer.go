package enforcer

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"kdeps/pkg/logging"
	"kdeps/pkg/schema"

	"github.com/spf13/afero"
)

// compareVersions compares two version strings and returns:
// -1 if v1 < v2
// 0 if v1 == v2
// 1 if v1 > v2
func compareVersions(v1, v2 string) (int, error) {
	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		v1Part, err1 := strconv.Atoi(v1Parts[i])
		v2Part, err2 := strconv.Atoi(v2Parts[i])

		if err1 != nil || err2 != nil {
			logging.Error("Invalid version format")
			return 0, errors.New("invalid version format")
		}

		if v1Part < v2Part {
			return -1, nil
		}
		if v1Part > v2Part {
			return 1, nil
		}
	}

	// If all parts compared are equal, return 0
	return 0, nil
}

// EnforceSchemaURL checks if the "amends" line contains the correct schema.kdeps.com/core URL
func EnforceSchemaURL(line string) error {
	if !strings.HasPrefix(line, "amends") {
		logging.Error("The .pkl file does not start with 'amends'")
		return errors.New("the pkl file does not start with 'amends'")
	}

	if !strings.Contains(line, "schema.kdeps.com/core") {
		logging.Error("The .pkl file does not contain 'schema.kdeps.com/core'")
		return errors.New("the pkl file does not contain 'schema.kdeps.com/core'")
	}

	return nil
}

// EnforcePklVersion extracts the version from the "amends" line and compares it with the provided schema version
func EnforcePklVersion(line, schemaVersion string) error {
	start := strings.Index(line, "@")
	end := strings.Index(line, "#")
	if start == -1 || end == -1 || start >= end {
		logging.Error("Invalid version format in the amends line")
		return errors.New("invalid version format in the amends line")
	}
	version := line[start+1 : end]

	comparison, err := compareVersions(version, schemaVersion)
	if err != nil {
		logging.Error("Version comparison error", "error", err)
		return err
	}

	if comparison == -1 {
		logging.Warn("Version in amends line is lower than schema version", "version", version, "schemaVersion", schemaVersion)
	} else if comparison == 1 {
		logging.Info("Version in amends line is higher than schema version", "version", version, "schemaVersion", schemaVersion)
	}

	return nil
}

// Helper function to get the keys of a map as a slice of strings
func validPklFilesKeys(validPklFiles map[string]bool) []string {
	keys := make([]string, 0, len(validPklFiles))
	for k := range validPklFiles {
		keys = append(keys, k)
	}
	return keys
}

func EnforcePklFilename(line string, filePath string) error {
	filename := strings.ToLower(filepath.Base(filePath))
	start := strings.Index(line, "#/")
	if start == -1 {
		logging.Error("Invalid format: could not extract .pkl filename")
		return errors.New("invalid format: could not extract .pkl filename")
	}
	pklFilename := line[start+2:]
	pklFilename = strings.Trim(pklFilename, `"`)

	logging.Info("Checking pkl filename", "line", line, "filePath", filePath, "pklFilename", pklFilename)

	validPklFiles := map[string]bool{
		"Kdeps.pkl":    true,
		"Workflow.pkl": true,
		"Resource.pkl": true,
	}

	if pklFilename == "Resource.pkl" && (filename == ".kdeps.pkl" || filename == "workflow.pkl") {
		logging.Error("Invalid filename for Resource.pkl", "filename", filename, "pklFilename", pklFilename)
		return errors.New("invalid filename for Resource.pkl")
	}

	if !validPklFiles[pklFilename] {
		logging.Error("Invalid .pkl file in amends line", "expected", validPklFilesKeys(validPklFiles), "found", pklFilename)
		return errors.New("invalid .pkl file in amends line")
	}

	var expectedPkl string
	var pklType string
	switch pklFilename {
	case "Kdeps.pkl":
		pklType = "configuration file"
		expectedPkl = ".kdeps.pkl"
	case "Workflow.pkl":
		pklType = "workflow file"
		expectedPkl = "workflow.pkl"
	}

	if expectedPkl != filename && pklFilename != "Resource.pkl" {
		logging.Error("Invalid .pkl filename", "expected", expectedPkl, "found", filename, "type", pklType)
		return errors.New("invalid .pkl filename for a " + pklType)
	}

	return nil
}

func EnforceFolderStructure(fs afero.Fs, filePath string) error {
	expectedFile := "workflow.pkl"
	expectedFolders := map[string]bool{
		"resources": false,
		"data":      false,
	}

	ignoredFiles := map[string]bool{
		".kdeps.pkl": true,
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		logging.Error("Error getting absolute path", "filePath", filePath, "error", err)
		return err
	}

	fileInfo, err := fs.Stat(absPath)
	if err != nil {
		logging.Error("Error reading file info", "filePath", filePath, "error", err)
		return err
	}

	var absTargetDir string
	if !fileInfo.IsDir() {
		absTargetDir = filepath.Dir(absPath)
	} else {
		absTargetDir = absPath
	}

	files, err := afero.ReadDir(fs, absTargetDir)
	if err != nil {
		logging.Error("Error reading directory contents", "dir", absTargetDir, "error", err)
		return err
	}

	for _, file := range files {
		if _, isIgnored := ignoredFiles[file.Name()]; isIgnored {
			logging.Info("Ignored file found", "file", file.Name())
			continue
		}

		if file.IsDir() {
			if _, ok := expectedFolders[file.Name()]; !ok {
				logging.Error("Unexpected folder found", "folder", file.Name())
				return errors.New("unexpected folder found: " + file.Name())
			}

			expectedFolders[file.Name()] = true

			if file.Name() == "resources" {
				err := enforceResourcesFolder(fs, filepath.Join(absTargetDir, "resources"))
				if err != nil {
					return err
				}
			}
		} else {
			if file.Name() != expectedFile {
				logging.Error("Unexpected file found", "file", file.Name())
				return errors.New("unexpected file found: " + file.Name())
			}
		}
	}

	for folder, found := range expectedFolders {
		if !found {
			logging.Warn("Folder does not exist", "folder", folder)
		}
	}

	return nil
}

func EnforceResourceRunBlock(fs afero.Fs, file string) error {
	// Load the .pkl file content as a string
	pklData, err := afero.ReadFile(fs, file)
	if err != nil {
		logging.Error("Failed to read .pkl file", "file", file, "error", err)
		return err
	}
	content := string(pklData)

	// Regular expressions to match exec, chat, and httpClient, focusing only on the start
	execRegex := regexp.MustCompile(`(?i)[\s\n]*exec\s*{`)
	chatRegex := regexp.MustCompile(`(?i)[\s\n]*chat\s*{`)
	httpClientRegex := regexp.MustCompile(`(?i)[\s\n]*httpClient\s*{`)

	// Check for matches
	execMatch := execRegex.MatchString(content)
	chatMatch := chatRegex.MatchString(content)
	httpClientMatch := httpClientRegex.MatchString(content)

	// Count how many are non-null
	countNonNull := 0
	if execMatch {
		countNonNull++
	}
	if chatMatch {
		countNonNull++
	}
	if httpClientMatch {
		countNonNull++
	}

	// If more than one is non-null, return an error
	if countNonNull > 1 {
		errMsg := fmt.Sprintf("Error: resources run block can only contain one of 'exec', 'chat', or 'httpClient'. Please create a new dedicated resource for the function. Found %d in file: %s", countNonNull, file)
		logging.Error(errMsg)
		return errors.New(errMsg)
	}

	logging.Info("Run block validated successfully", "file", file)
	return nil
}

func enforceResourcesFolder(fs afero.Fs, resourcesPath string) error {
	files, err := afero.ReadDir(fs, resourcesPath)
	if err != nil {
		logging.Error("Error reading resources folder", "path", resourcesPath, "error", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() && file.Name() == "external" {
			continue
		}

		if file.IsDir() {
			logging.Error("Unexpected directory in resources folder", "dir", file.Name())
			return errors.New("unexpected directory found in resources folder: " + file.Name())
		}

		if filepath.Ext(file.Name()) != ".pkl" {
			logging.Error("Unexpected file found in resources folder", "file", file.Name())
			return errors.New("unexpected file found in resources folder: " + file.Name())
		}

		if filepath.Ext(file.Name()) == ".pkl" {
			fullFilePath := filepath.Join(resourcesPath, file.Name())
			if err := EnforceResourceRunBlock(fs, fullFilePath); err != nil {
				logging.Error("Failed to process .pkl file", "file", fullFilePath, "error", err)
				return err
			}
		}
	}

	return nil
}

// EnforcePklTemplateAmendsRules combines the three validations (schema URL, version, and .pkl file)
func EnforcePklTemplateAmendsRules(fs afero.Fs, filePath string) error {
	// Open the file containing the amends line
	file, err := fs.Open(filePath)
	if err != nil {
		logging.Error("Failed to open file", "filePath", filePath, "error", err)
		return err
	}
	defer file.Close()

	// Create a new scanner to read the amends file line by line
	scanner := bufio.NewScanner(file)

	// Iterate over lines and skip empty or whitespace-only lines
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text()) // Remove leading and trailing whitespace
		if line == "" {
			continue // Skip empty lines
		}

		logging.Info("Processing line", "line", line)

		// Check if the file has a .pkl extension
		if filepath.Ext(file.Name()) != ".pkl" {
			logging.Error("Unexpected file type", "file", file.Name())
			return errors.New("unexpected file type: " + file.Name())
		}

		// Validate the line in stages
		if err := EnforceSchemaURL(line); err != nil {
			logging.Error("Schema URL validation failed", "line", line, "error", err)
			return err
		}

		if err := EnforcePklVersion(line, schema.SchemaVersion); err != nil {
			logging.Error("Version validation failed", "line", line, "error", err)
			return err
		}

		if err := EnforcePklFilename(line, filePath); err != nil {
			logging.Error("Filename validation failed", "line", line, "error", err)
			return err
		}

		// All checks passed
		logging.Info("All validations passed for the line", "line", line)
		return nil
	}

	// Check for any scanning error
	if err := scanner.Err(); err != nil {
		logging.Error("Error while scanning the file", "filePath", filePath, "error", err)
		return err
	}

	// Return error if no valid amends line was found
	logging.Error("No valid 'amends' line found in the file", "filePath", filePath)
	return errors.New("no valid 'amends' line found")
}
