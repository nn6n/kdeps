package evaluator_test

import (
	"fmt"
	"testing"

	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndProcessPklFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logging.CreateLogger()
	logger := logging.GetLogger()

	sections := []string{"section1", "section2"}
	finalFileName := "/tmp/final.pkl"
	pklTemplate := "Kdeps.pkl"

	processFunc := func(fs afero.Fs, tmpFile string, headerSection string, logger *logging.Logger) (string, error) {
		content, err := afero.ReadFile(fs, tmpFile)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s\n%s", headerSection, string(content)), nil
	}

	t.Run("CreateAndProcessAmends", func(t *testing.T) {
		err := evaluator.CreateAndProcessPklFile(fs, sections, finalFileName, pklTemplate, logger, processFunc, false)
		assert.NoError(t, err, "CreateAndProcessPklFile should not return an error")
		content, err := afero.ReadFile(fs, finalFileName)
		require.NoError(t, err, "Final file should be created successfully")
		assert.Contains(t, string(content), "amends", "Final file content should include 'amends'")
		assert.Contains(t, string(content), sections[0], "Final file content should include section1")
	})

	t.Run("CreateAndProcessExtends", func(t *testing.T) {
		err := evaluator.CreateAndProcessPklFile(fs, sections, finalFileName, pklTemplate, logger, processFunc, true)
		assert.NoError(t, err, "CreateAndProcessPklFile should not return an error")
		content, err := afero.ReadFile(fs, finalFileName)
		require.NoError(t, err, "Final file should be created successfully")
		assert.Contains(t, string(content), "extends", "Final file content should include 'extends'")
		assert.Contains(t, string(content), sections[1], "Final file content should include section2")
	})
}
