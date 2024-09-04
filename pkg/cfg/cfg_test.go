package cfg

import (
	"path/filepath"
	"testing"

	"github.com/cucumber/godog"
	"github.com/spf13/afero"
)

var (
	testFs         = afero.NewOsFs()
	currentDirPath string
	homeDirPath    string
	testConfigFile string
	fileThatExist  string
	testingT       *testing.T
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a file "([^"]*)" exists in the current directory$`, aFileExistsInTheCurrentDirectory)
			ctx.Step(`^a file "([^"]*)" exists in the home directory$`, aFileExistsInTheHomeDirectory)
			ctx.Step(`^the configuration file is "([^"]*)"$`, theConfigurationFileIs)
			ctx.Step(`^the configuration is loaded in the current directory$`, theConfigurationIsLoadedInTheCurrentDirectory)
			ctx.Step(`^the configuration is loaded in the home directory$`, theConfigurationIsLoadedInTheHomeDirectory)
			ctx.Step(`^the current directory is "([^"]*)"$`, theCurrentDirectoryIs)
			ctx.Step(`^the home directory is "([^"]*)"$`, theHomeDirectoryIs)
			ctx.Step(`^a file "([^"]*)" does not exists in the home or current directory$`, aFileDoesNotExistsInTheHomeOrCurrentDirectory)
			ctx.Step(`^the configuration fails to load any configuration$`, theConfigurationFailsToLoadAnyConfiguration)
			ctx.Step(`^the configuration file will be generated to "([^"]*)"$`, theConfigurationFileWillBeGeneratedTo)
			ctx.Step(`^the configuration will be edited$`, theConfigurationWillBeEdited)
			ctx.Step(`^the configuration will be validated$`, theConfigurationWillBeValidated)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aFileExistsInTheCurrentDirectory(arg1 string) error {
	doc := `
amends "package://schema.kdeps.com/core@0.0.25#/Kdeps.pkl"

dockerImage = "alpine:3.14"
llmSettings {
  llmAPIKeys {
    openai_api_key = null
    mistral_api_key = null
    huggingface_api_token = null
    groq_api_key = null
  }
  llmBackend = "ollama"
  llmModel = "llama3.1"
  modelFile = null
}
`
	file := filepath.Join(currentDirPath, arg1)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	fileThatExist = file

	return nil
}

func aFileExistsInTheHomeDirectory(arg1 string) error {
	doc := `
amends "package://schema.kdeps.com/core@0.0.25#/Kdeps.pkl"

dockerImage = "alpine:3.14"
llmSettings {
  llmAPIKeys {
    openai_api_key = null
    mistral_api_key = null
    huggingface_api_token = null
    groq_api_key = null
  }
  llmBackend = "ollama"
  llmModel = "llama3.1"
  modelFile = null
}
`
	file := filepath.Join(homeDirPath, arg1)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	fileThatExist = file

	return nil
}

func theConfigurationFileIs(arg1 string) error {
	if _, err := testFs.Stat(fileThatExist); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheCurrentDirectory() error {
	env := &Environment{
		Home: "",
		Pwd:  currentDirPath,
	}

	if err := FindConfiguration(testFs, env); err != nil {
		return err
	}

	if err := LoadConfiguration(testFs); err != nil {
		return err
	}

	return nil
}

func theConfigurationIsLoadedInTheHomeDirectory() error {
	env := &Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	if err := FindConfiguration(testFs, env); err != nil {
		return err
	}

	if err := LoadConfiguration(testFs); err != nil {
		return err
	}

	return nil
}

func theCurrentDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")

	if err != nil {
		return err
	}

	currentDirPath = tempDir

	return nil
}

func theHomeDirectoryIs(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", "")

	if err != nil {
		return err
	}

	homeDirPath = tempDir

	return nil
}

func aFileDoesNotExistsInTheHomeOrCurrentDirectory(arg1 string) error {
	fileThatExist = ""

	return nil
}

func theConfigurationFailsToLoadAnyConfiguration() error {
	env := &Environment{
		Home: homeDirPath,
		Pwd:  currentDirPath,
	}

	if err := FindConfiguration(testFs, env); err != nil {
		return err
	}

	return nil
}

func theConfigurationFileWillBeGeneratedTo(arg1 string) error {
	env := &Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	if err := GenerateConfiguration(testFs, env); err != nil {
		return err
	}

	if err := LoadConfiguration(testFs); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeEdited() error {
	env := &Environment{
		Home:           homeDirPath,
		Pwd:            "",
		NonInteractive: "1",
	}

	if err := EditConfiguration(testFs, env); err != nil {
		return err
	}

	return nil
}

func theConfigurationWillBeValidated() error {
	env := &Environment{
		Home: homeDirPath,
		Pwd:  "",
	}

	if err := ValidateConfiguration(testFs, env); err != nil {
		return err
	}

	return nil
}
