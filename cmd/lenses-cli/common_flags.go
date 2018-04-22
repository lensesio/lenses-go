package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/jmespath/go-jmespath"
	"gopkg.in/yaml.v2"
)

var (
	// if true then it doesn't prints json result(s) with indent.
	// Defaults to false.
	// It's not a global flag, but it's a common one, all commands that return results
	// use that via command flag binding.
	noPretty bool
	// if true then commands will not output info messages, like "Processor ___ created".
	// Look the `echo` func for more, it's not a global flag but it's a common one, all commands that return info messages
	// set that via command flag binding.
	//
	// Defaults to false.
	silent bool

	// jmespathQuery query to further filter any results, if any.
	// It's not a global flag, but it's a common one, all commands that return results
	// set that via command flag binding.
	jmespathQuery string
)

func echo(cmd *cobra.Command, /* io.Writer is more generic, let's make it explicit use within commands */
	format string, args ...interface{}) error {

	if silent {
		return nil
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n" // add a new line.
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	return err
}

/*
	Below we have two identical functions, keep them separate there are limits on the final output otherwise.
*/

// outlineStrings will accept a key, i.e "name" and entries i.e ["schema1", "schema2", "schema3"]
// and will convert it to a slice of [{"name":"schema1"},"name":"schema2", "name":"schema3"}] to be able to be printed via `printJSON`.
func outlineStringResults(key string, entries []string) (items []interface{}) { // why not? (items []map[string]string) because jmespath can't work with it, only with []interface.
	// key = strings.Title(key)
	for _, entry := range entries {
		items = append(items, map[string]string{key: entry})
	}

	return
}

// outlineStrings will accept a key, i.e "version" and entries i.e [1, 2, 3]
// and will convert it to a slice of [{"version":3},"version":1, "version":2}] to be able to be printed via `printJSON`.
func outlineIntResults(key string, entries []int) (items []interface{}) {
	// key = strings.Title(key)
	for _, entry := range entries {
		items = append(items, map[string]int{key: entry})
	}

	return
}

type transformer func([]byte, bool) ([]byte, error)

func toJSON(v interface{}, pretty bool, transformers ...transformer) ([]byte, error) {
	var (
		rawJSON []byte
		err     error
	)

	if pretty {
		rawJSON, err = DefaultTranscoder.EncodeIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
	} else {
		rawJSON, err = DefaultTranscoder.Encode(v)
		if err != nil {
			return nil, err
		}
	}

	for _, transformer := range transformers {
		if transformer == nil {
			continue // may give a nil transformer in variadic input.
		}
		b, err := transformer(rawJSON, pretty)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			continue
		}
		rawJSON = b
	}

	return rawJSON, err
}

func jmesQuery(query string, v interface{}) transformer {
	return func(rawJSON []byte, pretty bool) ([]byte, error) {
		if query == "" || strings.TrimSpace(string(rawJSON)) == "[]" { // if it's empty, exit.
			return nil, nil // don't throw error here, just skip it by returning nil result and nil error.
		}

		result, err := jmespath.Search(query, v)
		if err != nil {
			return nil, err
		}

		return toJSON(result, pretty)
	}
}

// based on global flags.
func printJSON(out io.Writer, v interface{}) error { //, pretty bool, transformers ...transformer) (err error) {
	rawJSON, err := toJSON(v, !noPretty, jmesQuery(jmespathQuery, v))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(rawJSON))
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
		return DefaultTranscoder.Decode(result, outPtr)
		// return fmt.Errorf("unsupported file type '%s', use .json for configs instead", ext)
	}

}

// readInPipe used at sql commands (simple and --validate)
// to read from the input pipe, but in the future may be used elsewhere.
//
// First argument returns true if in pipe has any data to read from,
// if false then the caller can continue by requiring a flag.
// Second argument returns the data of the io pipe,
// and third one is the error cames from .Stat() or from the ReadAll() of the in pipe.
func readInPipe() (bool, []byte, error) {
	// check if has data, otherwise it stucks.
	in := os.Stdin
	f, err := in.Stat()
	if err != nil {
		return false, nil, err
	}

	// check if has data is required, otherwise it stucks.
	if !(f.Mode()&os.ModeNamedPipe == 0) {
		b, err := ioutil.ReadAll(in)
		if err != nil {
			return true, nil, err
		}
		return true, b, nil
	}

	return false, nil, nil
}

type flags map[string]interface{}

// this function can be used to manually check for required flags, when the command does not specify a required flag (mostly because of file loading feature).
func checkRequiredFlags(cmd *cobra.Command, nameValuePairs flags) (err error) {
	if nameValuePairs == nil {
		return nil
	}

	var emptyFlags []string

	for name, value := range nameValuePairs {
		if reflect.TypeOf(value).Comparable() {
			if value == reflect.Zero(reflect.TypeOf(value)).Interface() {
				emptyFlags = append(emptyFlags, strconv.Quote(name))
			}
		}
	}

	if n := len(emptyFlags); n > 0 {
		if n == 1 {
			// required flag "flag 1" not set
			err = fmt.Errorf("required flag %s not set", emptyFlags[0])
		} else {
			// required flags "flag 1" and "flag 2" not set
			// required flags "flag 1", "flag 2" and "flag 3" not set
			err = fmt.Errorf("required flags %s and %s not set",
				strings.Join(emptyFlags[0:n-1], ", "), emptyFlags[n-1])
		}

		if len(nameValuePairs) == n {
			// if all required flags are not passed, then show an example in the end.
			err = fmt.Errorf("%s\nexample:\n\t%s", err, cmd.Example)
		}
	}

	return
}

// This is a self-crafted hack to convert custom types to a compatible cobra flag.
// Do NOT touch it.
//
// Supported custom types underline are: strings, ints and booleans only.
type flagVar struct {
	value reflect.Value
}

func newVarFlag(v interface{}) *flagVar {
	return &flagVar{reflect.ValueOf(v)}
}

func (f flagVar) String() string {
	return f.value.Elem().String()
}

func (f flagVar) Set(v string) error {
	typ := f.value.Elem().Kind()
	switch typ {
	case reflect.String:
		f.value.Elem().SetString(v)
		break
	case reflect.Int:
		intValue, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		f.value.Elem().SetInt(int64(intValue))
		break
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}

		f.value.Elem().SetBool(boolValue)
		break
	}

	return nil
}

func (f flagVar) Type() string {
	return f.value.Elem().Kind().String() // reflect/type.go#605
}
