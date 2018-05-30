package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

const headerTag = "header"

func getHeaders(typ reflect.Type) (headers []string) {
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)
		if header := f.Tag.Get(headerTag); header != "" {
			// header is the first part.
			headers = append(headers, strings.Split(header, ",")[0])
		}
	}

	return
}

func getRow(val reflect.Value) (rightCells []int, row []string) {
	v := reflect.Indirect(val)
	typ := v.Type()
	j := 0
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)
		if header := f.Tag.Get(headerTag); header != "" {
			fieldValue := reflect.Indirect(v.Field(i))

			if fieldValue.CanInterface() {
				s := ""
				vi := fieldValue.Interface()

				switch fieldValue.Kind() {
				case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
					rightCells = append(rightCells, j)

					sInt64, err := strconv.ParseInt(fmt.Sprintf("%d", vi), 10, 64)
					if err != nil || sInt64 == 0 {
						s = "0"
						break
					}

					s = nearestThousandFormat(float64(sInt64))
					break
				case reflect.Float32, reflect.Float64:
					s = fmt.Sprintf("%.2f", vi)
					rightCells = append(rightCells, j)
					break
				case reflect.Bool:
					if vi.(bool) {
						s = "Yes"
					} else {
						s = "No"
					}
					break
				case reflect.Slice, reflect.Array:
					rightCells = append(rightCells, j)

					// check the second part, if it's there then check for "len", if there then show the length,
					// otherwise split the slice into stringable entries.
					// the second part is used as an alternative printable string value if empty or nil.
					if h := strings.Split(header, ","); len(h) > 1 {
						if alternative := h[1]; alternative == "len" { // len is a static name, should cleanup the entire logic.
							s = strconv.Itoa(fieldValue.Len())
						} else {
							s = alternative
						}

						break
					}

					for fieldSliceIdx, fieldSliceLen := 0, fieldValue.Len(); fieldSliceIdx < fieldSliceLen; fieldSliceIdx++ {
						vf := fieldValue.Index(fieldSliceIdx)
						if vf.CanInterface() {
							s += fmt.Sprintf("%v", vf.Interface())
							if hasMore := fieldSliceIdx+1 > fieldSliceLen; hasMore {
								s += ", "
							}
						}
					}

					break
				default:
					s = fmt.Sprintf("%v", vi)
				}

				if s == "" {
					// the second part is used as an alternative printable string value if empty or nil.
					if h := strings.Split(header, ","); len(h) > 1 {
						s = h[1]
					}
				}

				row = append(row, s)
				j++
			}
		}
	}

	return
}

type rowFilter func(reflect.Value) bool

func canAcceptRow(in reflect.Value, filters []rowFilter) bool {
	acceptRow := true
	for _, filter := range filters {
		if !filter(in) {
			acceptRow = false
			break
		}
	}

	return acceptRow
}

func makeFilters(in reflect.Value, filters []interface{}) (f []rowFilter) {
	for _, filter := range filters {
		filterTyp := reflect.TypeOf(filter)
		// must be a function that accepts one input argument which is the same of the "v".
		if filterTyp.Kind() != reflect.Func || filterTyp.NumIn() != 1 /* not receiver */ || filterTyp.In(0) != in.Type() {
			continue
		}

		// must be a function that returns a single boolean value.
		if filterTyp.NumOut() != 1 || filterTyp.Out(0).Kind() != reflect.Bool {
			continue
		}

		filterValue := reflect.ValueOf(filter)
		func(filterValue reflect.Value) {
			f = append(f, func(in reflect.Value) bool {
				out := filterValue.Call([]reflect.Value{in})
				return out[0].Interface().(bool)
			})
		}(filterValue)
	}

	return

}

// Usage with filters:
// printTable(cmd, topics, func(t lenses.Topic) bool { /* or any type */
// 	return t.TopicName == "test" || t.TopicName == "moving_ships"
// })
func printTable(cmd *cobra.Command, v interface{}, filters ...interface{}) error {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	var (
		headers           []string
		rows              [][]string
		rightAligmentCols []int
	)

	if val := reflect.Indirect(reflect.ValueOf(v)); val.Kind() == reflect.Slice {
		var f []rowFilter
		for i, n := 0, val.Len(); i < n; i++ {
			v := val.Index(i)

			if i == 0 {
				// make filters once instead of each time for each entry, they all have the same v type.
				f = makeFilters(v, filters)
				headers = getHeaders(v.Type())
			}

			if !v.IsValid() {
				rows = append(rows, []string{""})
				continue
			}
			right, row := getRow(v)
			if i == 0 {
				rightAligmentCols = right
			}

			if canAcceptRow(v, f) {
				rows = append(rows, row)
			}
		}
	} else {
		// single.
		headers = getHeaders(val.Type())
		right, row := getRow(val)
		rightAligmentCols = right
		if canAcceptRow(val, makeFilters(val, filters)) {
			rows = append(rows, row)
		}

	}

	if len(headers) == 0 {
		return nil
	}

	// if more than 3 then show the length of results.
	if n := len(rows); n > 3 {
		headers[0] = fmt.Sprintf("%s (%d) ", headers[0], len(rows))
	}

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
	columnAlignment := make([]int, len(rows), len(rows))
	for i := range columnAlignment {
		columnAlignment[i] = tablewriter.ALIGN_LEFT

		for _, j := range rightAligmentCols {
			if i == j {
				columnAlignment[i] = tablewriter.ALIGN_RIGHT
				break
			}
		}

	}

	table.SetColumnAlignment(columnAlignment)

	fmt.Fprintln(cmd.OutOrStdout())
	table.Render()

	return nil
}
