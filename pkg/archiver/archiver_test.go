package archiver

import (
	"errors"
	"fmt"
	"kdeps/pkg/enforcer"
	"kdeps/pkg/resource"
	"kdeps/pkg/workflow"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cucumber/godog"
	"github.com/kr/pretty"
	"github.com/spf13/afero"
)

var (
	testFs             = afero.NewOsFs()
	testingT           *testing.T
	aiAgentDir         string
	resourcesDir       string
	dataDir            string
	workflowFile       string
	resourceFile       string
	kdepsDir           string
	projectDir         string
	packageDir         string
	lastCreatedPackage string
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: func(ctx *godog.ScenarioContext) {
			ctx.Step(`^a kdeps archive "([^"]*)" is opened$`, aKdepsArchiveIsOpened)
			ctx.Step(`^an ai agent on "([^"]*)" folder exists$`, anAiAgentOnFolder)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)"$`, itHasAWorkflowFile)
			ctx.Step(`^the content of that archive file will be extracted to "([^"]*)"$`, theContentOfThatArchiveFileWillBeExtractedTo)
			ctx.Step(`^the pkl files is valid$`, thePklFilesIsValid)
			ctx.Step(`^the project is valid$`, theProjectIsValid)
			ctx.Step(`^the project will be archived to "([^"]*)"$`, theProjectWillBeArchivedTo)
			ctx.Step(`^the "([^"]*)" system folder exists$`, theSystemFolderExists)
			ctx.Step(`^theres a data file$`, theresADataFile)
			ctx.Step(`^the pkl files is invalid$`, thePklFilesIsInvalid)
			ctx.Step(`^the project is invalid$`, theProjectIsInvalid)
			ctx.Step(`^the project will not be archived to "([^"]*)"$`, theProjectWillNotBeArchivedTo)

			ctx.Step(`^it has a "([^"]*)" file with id property "([^"]*)" and dependent on "([^"]*)"$`, itHasAFileWithIdPropertyAndDependentOn)
			ctx.Step(`^it has a "([^"]*)" file with no dependency with id property "([^"]*)"$`, itHasAFileWithNoDependencyWithIdProperty)
			ctx.Step(`^it will be stored to "([^"]*)"$`, itWillBeStoredTo)
			ctx.Step(`^the project is compiled$`, theProjectIsCompiled)
			ctx.Step(`^the resource id for "([^"]*)" will be "([^"]*)" and dependency "([^"]*)"$`, theResourceIdForWillBeAndDependency)
			ctx.Step(`^the resource id for "([^"]*)" will be rewritten to "([^"]*)"$`, theResourceIdForWillBeRewrittenTo)
			ctx.Step(`^the workflow action configuration will be rewritten to "([^"]*)"$`, theWorkflowActionConfigurationWillBeRewrittenTo)
			ctx.Step(`^the resources and data folder exists$`, theResourcesAndDataFolderExists)
			ctx.Step(`^the data files will be copied to "([^"]*)"$`, theDataFilesWillBeCopiedTo)
			ctx.Step(`^the package file "([^"]*)" will be created$`, thePackageFileWillBeCreated)
			ctx.Step(`^it has a workflow file that has name property "([^"]*)" and version property "([^"]*)" and default action "([^"]*)" and workspaces "([^"]*)"$`, itHasAWorkflowFileDependencies)
			ctx.Step(`^the resource file "([^"]*)" exists in the "([^"]*)" agent "([^"]*)"$`, theResourceFileExistsInTheAgent)
		},
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/archiver"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	testingT = t

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func aKdepsArchiveIsOpened(arg1 string) error {
	name, version := regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[1], regexp.MustCompile(`^([a-zA-Z]+)-([\d]+\.[\d]+\.[\d]+)\.kdeps$`).FindStringSubmatch(arg1)[2]

	kdepsAgentPath := filepath.Join(kdepsDir, "agents/"+name+"/"+version)
	if _, err := testFs.Stat(kdepsAgentPath); err == nil {
		return errors.New("agent should not yet exists on system agents dir")
	}

	proj, err := ExtractPackage(testFs, kdepsDir, lastCreatedPackage)
	if err != nil {
		return err
	}

	fmt.Printf("%# v", pretty.Formatter(proj))

	return nil
}

func theSystemFolderExists(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	kdepsDir = tempDir

	packageDir = kdepsDir + "/packages"
	if err := testFs.MkdirAll(packageDir, 0755); err != nil {
		return err
	}

	return nil
}

func anAiAgentOnFolder(arg1 string) error {
	tempDir, err := afero.TempDir(testFs, "", arg1)
	if err != nil {
		return err
	}

	aiAgentDir = tempDir

	return nil
}

func itHasAFileWithIdPropertyAndDependentOn(arg1, arg2, arg3 string) error {
	// Check if arg3 is a CSV (contains commas)
	var requiresSection string
	if strings.Contains(arg3, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg3, ",")
		var requiresLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			requiresLines = append(requiresLines, fmt.Sprintf(`  "%s"`, value))
		}
		requiresSection = "requires {\n" + strings.Join(requiresLines, "\n") + "\n}"
	} else {
		// Single value case
		requiresSection = fmt.Sprintf(`requires {
  "%s"
}`, arg3)
	}

	// Create the document with the id and requires block
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Resource.pkl"

id = "%s"
%s
`, arg2, requiresSection)

	// Write to the file
	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	resourceFile = file

	return nil
}

func itWillBeStoredTo(arg1 string) error {
	workflowFile = filepath.Join(kdepsDir, arg1)

	if _, err := testFs.Stat(workflowFile); err != nil {
		return err
	}

	return nil
}

func theProjectIsCompiled() error {
	wf, err := workflow.LoadWorkflow(workflowFile)
	if err != nil {
		return err
	}

	projectDir, _, _ := CompileProject(testFs, wf, kdepsDir, aiAgentDir)

	workflowFile = filepath.Join(projectDir, "workflow.pkl")

	return nil
}

func theResourceIdForWillBeAndDependency(arg1, arg2, arg3 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(resFile)
		if err != nil {
			return err
		}
		if res.Id != arg2 {
			return errors.New("Should be equal!")
		}
		found := false
		for _, v := range *res.Requires {
			if v == arg3 {
				found = true
				break
			}
		}

		if !found {
			return errors.New("Require found!")
		}
	}

	return nil
}

func theResourceIdForWillBeRewrittenTo(arg1, arg2 string) error {
	resFile := filepath.Join(projectDir, "resources/"+arg1)
	if _, err := testFs.Stat(resFile); err == nil {
		res, err := resource.LoadResource(resFile)
		if err != nil {
			return err
		}

		if res.Id != arg2 {
			return errors.New("Should be equal!")
		}
	}

	return nil
}

func theWorkflowActionConfigurationWillBeRewrittenTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(workflowFile)
	if err != nil {
		return err
	}

	if wf.Action != arg1 {
		return errors.New(fmt.Sprintf("%s = %s does not match!", wf.Action, arg1))
	}

	return nil
}

func theResourcesAndDataFolderExists() error {
	resourcesDir = filepath.Join(aiAgentDir, "resources")
	if err := testFs.MkdirAll(resourcesDir, 0755); err != nil {
		return err
	}

	dataDir = filepath.Join(aiAgentDir, "data")
	if err := testFs.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	return nil
}

func itHasAFileWithNoDependencyWithIdProperty(arg1, arg2 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Resource.pkl"

id = "%s"
`, arg2)

	file := filepath.Join(resourcesDir, arg1)

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	resourceFile = file

	return nil
}

func itHasAWorkflowFile(arg1, arg2, arg3 string) error {
	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Workflow.pkl"

action = "%s"
name = "%s"
description = "My awesome AI Agent"
version = "%s"
`, arg3, arg1, arg2)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	workflowFile = file

	return nil
}

func theContentOfThatArchiveFileWillBeExtractedTo(arg1 string) error {
	fpath := filepath.Join(kdepsDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("there should be an agent dir present, but none was found")
	}

	return nil
}

func thePklFilesIsValid() error {
	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowFile, schemaVersionFilePath); err != nil {
		return err
	}

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, resourceFile, schemaVersionFilePath); err != nil {
		return err
	}

	return nil
}

func theProjectIsValid() error {
	if err := enforcer.EnforceFolderStructure(testFs, workflowFile); err != nil {
		return err
	}

	return nil
}

func theProjectWillBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(workflowFile)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, wf, kdepsDir, aiAgentDir)
	if err != nil {
		return err
	}

	if _, err := testFs.Stat(fpath); err != nil {
		return err
	}

	return nil
}

func theresADataFile() error {
	doc := "THIS IS A TEXT FILE: "

	for x := 0; x < 10; x++ {
		num := strconv.Itoa(x)
		file := filepath.Join(dataDir, fmt.Sprintf("textfile-%s.txt", num))

		f, _ := testFs.Create(file)
		f.WriteString(doc + num)
		f.Close()
	}

	return nil
}

func theDataFilesWillBeCopiedTo(arg1 string) error {
	file := filepath.Join(kdepsDir, arg1+"/textfile-1.txt")

	if _, err := testFs.Stat(file); err != nil {
		return err
	}

	return nil
}

func thePklFilesIsInvalid() error {
	doc := `
name = "invalid agent"
description = "a not valid configuration"
version = "five"
action = "hello World"
`
	file := filepath.Join(aiAgentDir, "workflow1.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	workflowFile = file

	if err := enforcer.EnforcePklTemplateAmendsRules(testFs, workflowFile, schemaVersionFilePath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectIsInvalid() error {
	if err := enforcer.EnforceFolderStructure(testFs, workflowFile); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func theProjectWillNotBeArchivedTo(arg1 string) error {
	wf, err := workflow.LoadWorkflow(workflowFile)
	if err != nil {
		return err
	}

	fpath, err := PackageProject(testFs, wf, kdepsDir, aiAgentDir)
	if err == nil {
		return errors.New("expected an error, but got nil")
	}

	if _, err := testFs.Stat(fpath); err == nil {
		return errors.New("expected an error, but got nil")
	}

	return nil
}

func thePackageFileWillBeCreated(arg1 string) error {
	fpath := filepath.Join(packageDir, arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}
	lastCreatedPackage = fpath

	return nil
}

func itHasAWorkflowFileDependencies(arg1, arg2, arg3, arg4 string) error {
	var workflowsSection string
	if strings.Contains(arg4, ",") {
		// Split arg3 into multiple values if it's a CSV
		values := strings.Split(arg4, ",")
		var workflowsLines []string
		for _, value := range values {
			value = strings.TrimSpace(value) // Trim any leading/trailing whitespace
			workflowsLines = append(workflowsLines, fmt.Sprintf(`  "%s"`, value))
		}
		workflowsSection = "workflows {\n" + strings.Join(workflowsLines, "\n") + "\n}"
	} else {
		// Single value case
		workflowsSection = fmt.Sprintf(`workflows {
  "%s"
}`, arg4)
	}

	doc := fmt.Sprintf(`
amends "package://schema.kdeps.com/core@0.0.34#/Workflow.pkl"

action = "%s"
name = "%s"
description = "My awesome AI Agent"
version = "%s"
%s
`, arg3, arg1, arg2, workflowsSection)

	file := filepath.Join(aiAgentDir, "workflow.pkl")

	f, _ := testFs.Create(file)
	f.WriteString(doc)
	f.Close()

	workflowFile = file

	return nil
}

func theResourceFileExistsInTheAgent(arg1, arg2, arg3 string) error {
	fpath := filepath.Join(kdepsDir, "agents/"+arg2+"/1.0.0/resources/"+arg1)
	if _, err := testFs.Stat(fpath); err != nil {
		return errors.New("expected a package, but got none")
	}

	return nil
}
