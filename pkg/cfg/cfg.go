package cfg

import (
	"context"
	"errors"
	"fmt"
	"kdeps/pkg/download"
	"os"
	"os/exec"
	"path/filepath"

	env "github.com/Netflix/go-env"
	execute "github.com/alexellis/go-execute/v2"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/editor"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
)

var (
	SystemConfigFileName = ".kdeps.pkl"
	ConfigFile           string
	HomeConfigFile       string
	CwdConfigFile        string
)

type Environment struct {
	Home           string `env:"HOME"`
	Pwd            string `env:"PWD"`
	NonInteractive string `env:"NON_INTERACTIVE,default=0"`
	Extras         env.EnvSet
}

func FindPklBinary() {
	binaryName := "pkl"
	if _, err := exec.LookPath(binaryName); err != nil {
		log.Fatalf(fmt.Sprintf("The binary '%s' does not exist in PATH. For more information, see: https://pkl-lang.org/\n", binaryName))
		os.Exit(1)
	}
}

func FindConfiguration(fs afero.Fs, environment *Environment) error {
	FindPklBinary()

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
		return nil
	}

	if len(environment.Pwd) > 0 {
		CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)

		ConfigFile = CwdConfigFile
		return nil
	}

	es, err := env.UnmarshalFromEnviron(&environment)
	if err != nil {
		return err
	}

	environment.Extras = es

	CwdConfigFile = filepath.Join(environment.Pwd, SystemConfigFileName)
	HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

	if _, err = fs.Stat(CwdConfigFile); err == nil {
		ConfigFile = CwdConfigFile
	} else if _, err = fs.Stat(HomeConfigFile); err == nil {
		ConfigFile = HomeConfigFile
	}

	if _, err = fs.Stat(ConfigFile); err == nil {
		log.Info("Configuration file found:", "config-file", ConfigFile)
	} else {
		log.Warn("Configuration file not found:", "config-file", ConfigFile)
	}

	return nil
}

func DownloadConfiguration(fs afero.Fs, environment *Environment) error {
	var skipPrompts bool = false

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if environment.NonInteractive == "1" {
		skipPrompts = true
	}

	if _, err := fs.Stat(ConfigFile); err != nil {
		var confirm bool
		if !skipPrompts {
			if err := huh.Run(
				huh.NewConfirm().
					Title("Configuration file not found. Do you want to generate one?").
					Description("The configuration will be validated. This will require the `pkl` package to be installed. Please refer to https://pkl-lang.org for more details.").
					Value(&confirm),
			); err != nil {
				return errors.New(fmt.Sprintln("Could not create a configuration file:", ConfigFile))
			}

			if !confirm {
				return errors.New("Aborted by user")
			}
		}
		download.DownloadFile(fs, "https://github.com/kdeps/schema/releases/latest/download/kdeps.pkl", ConfigFile)
	}

	return nil
}

func EditConfiguration(fs afero.Fs, environment *Environment) error {
	var skipPrompts bool = false

	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if environment.NonInteractive == "1" {
		skipPrompts = true
	}

	if _, err := fs.Stat(ConfigFile); err == nil {
		if !skipPrompts {
			c, err := editor.Cmd("kdeps", ConfigFile)
			if err != nil {
				return errors.New(fmt.Sprintln("Config file does not exist!"))
			}

			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				return errors.New(fmt.Sprintf("Missing %s.", "$EDITOR"))
			}
		}
	}

	return nil
}

func ValidateConfiguration(fs afero.Fs, environment *Environment) error {
	if len(environment.Home) > 0 {
		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	} else {
		es, err := env.UnmarshalFromEnviron(&environment)
		if err != nil {
			return err
		}

		environment.Extras = es

		HomeConfigFile = filepath.Join(environment.Home, SystemConfigFileName)

		ConfigFile = HomeConfigFile
	}

	if _, err := fs.Stat(ConfigFile); err == nil {
		cmd := execute.ExecTask{
			Command:     "pkl",
			Args:        []string{"eval", ConfigFile},
			StreamStdio: false,
		}

		res, err := cmd.Execute(context.Background())
		if err != nil {
			panic(err)
		}

		if res.ExitCode != 0 {
			panic("Non-zero exit code: " + res.Stderr)
		}
	}

	return nil
}

func LoadConfiguration(fs afero.Fs) error {
	log.Info("Reading config file:", "config-file", ConfigFile)

	_, err := kdeps.LoadFromPath(context.Background(), ConfigFile)
	if err != nil {
		return errors.New(fmt.Sprintf("Error reading config-file '%s': %s", ConfigFile, err))
	}
	return nil
}
