package main

import (
	"fmt"
	"reflect"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const headerTag = "header"

func getHeaders(typ reflect.Type) (headers []string) {
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)
		if header := f.Tag.Get(headerTag); header != "" {
			headers = append(headers, header)
		}
	}

	return
}

func getRow(val reflect.Value) (row []string) {
	v := reflect.Indirect(val)
	typ := v.Type()

	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)
		if header := f.Tag.Get(headerTag); header != "" {
			fieldValue := reflect.Indirect(v.Field(i))
			if fieldValue.CanInterface() {
				s := ""
				vi := fieldValue.Interface()
				switch fieldValue.Kind() {
				case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
					s = fmt.Sprintf("%d", vi)
					break
				case reflect.Float32, reflect.Float64:
					s = fmt.Sprintf("%.2f", vi)
					break
				case reflect.Bool:
					if vi.(bool) {
						s = "Yes"
					} else {
						s = "No"
					}
				default:
					s = fmt.Sprintf("%v", vi)
				}

				row = append(row, s)

			}
		}
	}

	return
}

func printTable(cmd *cobra.Command, v interface{}) error {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	var (
		headers []string
		rows    [][]string
	)

	if val := reflect.Indirect(reflect.ValueOf(v)); val.Kind() == reflect.Slice {
		for i, n := 0, val.Len(); i < n; i++ {
			v := val.Index(i)
			if i == 0 {
				headers = getHeaders(v.Type())
			}

			if !v.IsValid() {
				rows = append(rows, []string{""})
				continue
			}

			rows = append(rows, getRow(v))
		}
	} else {
		// single.
		headers = getHeaders(val.Type())
		rows = append(rows, getRow(val))
	}

	if len(headers) == 0 {
		return nil
	}

	headers[0] = fmt.Sprintf("%s (%d) ", headers[0], len(rows))
	table.SetHeader(headers)
	table.AppendBulk(rows)

	table.SetAutoFormatHeaders(true)
	table.SetAutoWrapText(true)
	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetColumnSeparator(" ")
	table.SetNewLine("\n")
	table.SetCenterSeparator(" ")

	fmt.Fprintln(cmd.OutOrStdout())
	table.Render()

	return nil
}
