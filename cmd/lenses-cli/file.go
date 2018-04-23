package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type (
	fileLoader struct {
		elseFunc     func() error
		pathResolver pathResolver
	}

	pathResolver func(cmd *cobra.Command, args []string) string
)

func (fl *fileLoader) Else(fn func() error) *fileLoader {
	fl.elseFunc = fn
	return fl
}

func (fl *fileLoader) WithPathResolve(fn pathResolver) *fileLoader {
	fl.pathResolver = fn
	return fl
}

func shouldTryLoadFile(cmd *cobra.Command, outPtr interface{}) *fileLoader {
	if reflect.TypeOf(outPtr).Kind() != reflect.Ptr {
		panic("outPtr is not a pointer")
	}

	fl := new(fileLoader)
	fl.pathResolver = func(c *cobra.Command, args []string) string {
		if len(args) == 0 {
			return ""
		}

		return args[0]
	}

	oldRunE := cmd.RunE

	cmd.RunE = func(c *cobra.Command, args []string) error {
		if path := fl.pathResolver(c, args); path != "" {
			if err := loadFile(c, path, outPtr); err != nil {
				return err
			}
		} else {
			if fl.elseFunc != nil {
				if err := fl.elseFunc(); err != nil {
					return err
				}
			}
		}

		return oldRunE(c, args)
	}

	return fl
}

// loadFile same as `tryReadFile` but it should be used for operations that we read the whole object from file,
// not just a sub property of it like `--config ./configs.json`.
//
// It just prints a message to the user that we load from file, so we ignore the flags.
func loadFile(cmd *cobra.Command, path string, outPtr interface{}) error {
	if err := echo(cmd, "Loading from file '%s', ignore flags", path); err != nil {
		return err
	}

	return tryReadFile(path, outPtr)
}

// tryReadFile will try to check if a flag value begins with 'flagFilePrefix'
// if so, then it will json parse its contents, decode them and set to the `outPtr`,
// otherwise it will decode the flagvalue using json unmarshaler and send the result to the `outPtr`.
func tryReadFile(flagValue string, outPtr interface{}) (err error) {
	result, err := tryReadFileContents(flagValue)
	if err != nil {
		return err
	}

	ext := filepath.Ext(flagValue)
	switch ext {
	case ".yml", ".yaml":
		return yaml.Unmarshal(result, outPtr)
	default:
		return json.Unmarshal(result, outPtr)
	}
}

func allowEmptyFlag(err error) error {
	if err == nil || err.Error() == errFlagMissing.Error() {
		return nil
	}

	return err
}

const flagFilePrefix = '@'

var errFlagMissing = fmt.Errorf("flag value is missing")

// tryReadFileContents will try to check if a flag value begins with 'flagFilePrefix'
// if so then it returns the contents of the filename given from the flagValue after the 'flagFilePrefix' character.
// Otherwise returns the flagValue as raw slice of bytes.
func tryReadFileContents(flagValue string) ([]byte, error) {
	if len(flagValue) == 0 {
		return nil, errFlagMissing
	}

	pathname := flagValue

	// check if argument is just a filepath and file exists,
	// if not then check if argument starts with @,
	// if so then this is the filepath, may relative, make it absolute if needed
	// and set the pathname to he corresponding value.
	if _, err := os.Stat(pathname); err != nil {
		if flagValue[0] != flagFilePrefix {
			// if file doesn't exist and argument doesn't start with @,
			// then return the flag value as raw bytes (the expected behavior if filepath not given).
			return []byte(flagValue), nil
		}

		pathname = flagValue[1:]
		if !filepath.IsAbs(pathname) {
			if abspath, err := filepath.Abs(pathname); err == nil {
				pathname = abspath
			}
		}
	}

	return ioutil.ReadFile(pathname)
}
