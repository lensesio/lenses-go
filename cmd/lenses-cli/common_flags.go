package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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

func newFlagSet(name string) *pflag.FlagSet {
	return pflag.NewFlagSet(name, pflag.ExitOnError)
}

func shouldCheckRequiredFlags(cmd *cobra.Command, nameValuesGetter func() flags) {
	oldRunE := cmd.RunE

	cmd.RunE = func(c *cobra.Command, args []string) error {
		if err := checkRequiredFlags(c, nameValuesGetter()); err != nil {
			return err
		}

		return oldRunE(c, args)
	}
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

func visitChildren(root *cobra.Command, visitor func(*cobra.Command)) *cobra.Command {
	if root.HasSubCommands() {
		for _, cmd := range root.Commands() {
			visitor(cmd)
		}
	}

	return root
}
