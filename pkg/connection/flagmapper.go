package connection

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/google/uuid"
	cobra "github.com/spf13/cobra"
)

type flagkind int

const (
	fstring flagkind = iota
	fint
	fbool
	fstringslice
	fkvmap
	ffileid
)

type flag struct {
	fieldname string
	flagname  string
	kind      flagkind
	ptr       bool
	val       interface{}
	dest      reflect.Value
	required  bool
}

// flagMapper helps with mapping Cobra flags onto a struct. Its functionality
// is rather specific, the implementation is fragile and not too elegant.
type flagMapper struct {
	cmd        *cobra.Command
	opts       FlagMapperOpts
	uploadFile uploadFunc
	flags      []flag
}

// uploadFunc uploads the given file and returns the assigned UUID. It is called
// when mapping a flag onto a "fileRef" structure.
type uploadFunc func(path string) (uuid.UUID, error)

type FlagMapperOpts struct {
	Descriptions map[string]string // Field name to description.
	Defaults     map[string]string // Field name to default, only for strings.
	Rename       map[string]string // Field name to flag name.
	Hide         []string          // Ignore those fields names.
	hideLut      map[string]bool   // Populated via hide.
}

func NewFlagMapper(cmd *cobra.Command, onto any, uploadFile uploadFunc, opts FlagMapperOpts) *flagMapper {
	ty := reflect.TypeOf(onto)
	if ty.Kind() != reflect.Ptr || ty.Elem().Kind() != reflect.Struct {
		panic("need &struct{}")
	}
	if opts.Defaults == nil {
		opts.Defaults = map[string]string{}
	}
	if opts.Descriptions == nil {
		opts.Descriptions = map[string]string{}
	}
	if opts.Rename == nil {
		opts.Rename = map[string]string{}
	}
	opts.hideLut = map[string]bool{}
	for _, v := range opts.Hide {
		opts.hideLut[v] = true
	}
	p := &flagMapper{
		cmd:        cmd,
		opts:       opts,
		uploadFile: uploadFile,
	}
	p.registerFlags(onto)
	return p
}

// MapFlags performs the mapping of the values of the previously registered
// flags onto the destination object. Whereas the registration has to be done
// while defining the Cobra command, MapFlags is to be ran while (pre-) running
// the command.
func (p *flagMapper) MapFlags() error {
	for _, f := range p.flags {
		_, hasDefault := p.opts.Defaults[f.fieldname] // if there is a default or change then assign.
		if !p.cmd.Flags().Lookup(f.flagname).Changed && !hasDefault {
			continue
		}
		switch f.kind {
		case fstring:
			if f.ptr {
				f.dest.Set(reflect.ValueOf(f.val.(*string)))
			} else {
				f.dest.SetString(*f.val.(*string))
			}
		case fint:
			if f.ptr {
				f.dest.Set(reflect.ValueOf(f.val.(*int)))
			} else {
				f.dest.SetInt(int64(*f.val.(*int)))
			}
		case fbool:
			if f.ptr {
				f.dest.Set(reflect.ValueOf(f.val.(*bool)))
			} else {
				f.dest.SetBool(*f.val.(*bool))
			}
		case fstringslice:
			f.dest.Set(reflect.ValueOf(*f.val.(*[]string)))
		case fkvmap:
			mm := map[string]string{}
			for _, v := range *f.val.(*[]string) {
				l, r, ok := strings.Cut(v, "=")
				if !ok {
					return fmt.Errorf("flag %q must have format: key=value, not: %q", f.flagname, v)
				}
				mm[l] = r
			}
			f.dest.Set(reflect.ValueOf(mm))
		case ffileid:
			sv := *f.val.(*string)
			var u uuid.UUID
			var err error
			if strings.HasPrefix(sv, "@") {
				fn := strings.TrimPrefix(sv, "@")
				if u, err = p.uploadFile(fn); err != nil {
					return fmt.Errorf("upload file %q: %w", fn, err)
				}
			} else {
				if u, err = uuid.Parse(sv); err != nil {
					return fmt.Errorf("parse uuid %q: %w", sv, err)
				}
			}
			dest := f.dest.Type()
			if f.ptr {
				dest = dest.Elem()
			}
			v := reflect.New(dest)                    // this makes the struct
			v.Elem().Field(0).Set(reflect.ValueOf(u)) // this sets the uuid field.
			if f.ptr {
				f.dest.Set(v)
			} else {
				f.dest.Set(v.Elem())
			}
		}
	}
	return nil
}

func (p *flagMapper) registerFlags(onto any) {
	p.cmd.Flags().SortFlags = false // Our structs are sorted by: required a-z; optional a-z.
	ty := reflect.ValueOf(onto).Elem()
	flags := p.trav(ty)

	for i, f := range flags {
		desc := p.opts.Descriptions[f.fieldname]
		if f.required {
			desc = "Required. " + desc
		} //else {
		// desc = "Optional. " + desc
		// }
		switch f.kind {
		case fstring:
			f.val = p.cmd.Flags().String(f.flagname, p.opts.Defaults[f.fieldname], desc)
		case fint:
			f.val = p.cmd.Flags().Int(f.flagname, 0, desc)
		case fbool:
			f.val = p.cmd.Flags().Bool(f.flagname, false, desc)
		case fstringslice:
			f.val = p.cmd.Flags().StringArray(f.flagname, nil, desc+" Specify multiple items by repeating the flag.")
		case fkvmap:
			f.val = p.cmd.Flags().StringArray(f.flagname, nil, desc+" Key/value pairs are to be given as key=value, repeating the flag for each pair.")
		case ffileid:
			f.val = p.cmd.Flags().String(f.flagname, "", desc+" Specify a filename prefixed with @ or a UUID.")
		}
		if f.required {
			p.cmd.MarkFlagRequired(f.flagname)
		}
		flags[i] = f
	}
	p.flags = flags
}

func (p *flagMapper) trav(v reflect.Value) []flag {
	ty := v.Type()
	var flags []flag
	for i := 0; i < ty.NumField(); i++ {
		f := ty.Field(i)
		if p.opts.hideLut[f.Name] {
			continue
		}
		fty := f.Type
		ptr := false
		if fty.Kind() == reflect.Ptr {
			fty = fty.Elem()
			ptr = true
		}
		required := !strings.Contains(f.Tag.Get("json"), "omitempty") // the "ptr==optional" holds not for slices :(,
		if f.Name == "Tags" {                                         // Hack. Lenses incorrectly rejects missing tags.
			required = false
			v.Field(i).Set(reflect.ValueOf([]string{})) // this is really bad.
		}
		if _, ok := p.opts.Defaults[f.Name]; ok {
			required = false
		}
		mf := flag{
			fieldname: f.Name,
			flagname:  kebabCase(f.Name),
			dest:      v.Field(i),
			required:  required,
			ptr:       ptr,
		}
		if n := p.opts.Rename[f.Name]; n != "" {
			mf.flagname = n
		}
		switch fty.Kind() {
		case reflect.String:
			mf.kind = fstring
		case reflect.Int:
			mf.kind = fint
		case reflect.Bool:
			mf.kind = fbool
		case reflect.Slice:
			if fty.Elem().Kind() != reflect.String { // []string
				panic(fty.Kind())
			}
			mf.kind = fstringslice
		case reflect.Map: // map[string]string
			if fty.Elem().Kind() != reflect.String || fty.Key().Kind() != reflect.String {
				panic("bad map")
			}
			mf.kind = fkvmap
		case reflect.Struct:
			if fty.NumField() == 1 && fty.Field(0).Type.Name() == "UUID" && strings.Contains(fty.Field(0).Tag.Get("json"), "fileId") { // not watertight but hey.
				mf.kind = ffileid
				break
			}
			if ptr {
				panic(fty.Name())
			}
			flags = append(flags, p.trav(v.Field(i))...)
			continue
		default:
			panic(f.Name)
		}
		flags = append(flags, mf)
	}
	return flags
}

func kebabCase(s string) string {
	s = strings.ReplaceAll(s, "HTTP", "Http") // hack hack hack...
	s = strings.ReplaceAll(s, "URL", "Url")
	s = strings.ReplaceAll(s, "AWS", "Aws")
	prevU := false
	o := ""
	for i, r := range s {
		curU := unicode.IsUpper(r)
		if curU && !prevU && i != 0 {
			o += "-"
		}
		o += string(unicode.ToLower(r))
		prevU = curU
	}
	return o
}
