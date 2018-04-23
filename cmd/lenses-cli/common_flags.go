package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

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
