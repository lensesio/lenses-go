package connection

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// connCrud aims to kill repetitive boilerplate code -- at the cost of increased
// complexity. The connection CRUD operations have a similiar structure. A
// connCrud is an intermediate step. Each C/R/U/D operation is represented by a
// single connCrud. The connCruds are then translated into cobra.Commands.
type connCrud struct {
	use   string         // => cobra.Command.Use.
	short string         // => cobra.Command.Short.
	opts  FlagMapperOpts // How to perform the flag mapping...
	onto  any            // .. onto this object.
	// some CRUD operations receive no argument, others do, some return a value,
	// others don't, etc. Each operation's logic is placed into the most
	// appropriate run* function and executes during command.RunE().
	runWithargNoRet func(arg string) error
	runWithArgRet   func(arg string) (interface{}, error)
	runNoArgRet     func() (interface{}, error)
	// Some Lenses connections assume a default name, e.g. "kafka". Be nice to
	// our users and default to it where possible.
	defaultArg string
}

func connCrudsToCobra(connName string, up uploadFunc, ms ...connCrud) (cobs []*cobra.Command) {
	cobs = make([]*cobra.Command, len(ms))
	for i, m := range ms {
		cobs[i] = connCrudToCobra(connName, up, m)
	}
	return cobs
}

func connCrudToCobra(connName string, up uploadFunc, m connCrud) *cobra.Command {
	cmd := &cobra.Command{
		Use:              m.use,
		Short:            m.short,
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	if cmd.Use == "upsert" {
		cmd.Aliases = append(cmd.Aliases, "update", "create")
	}
	short := map[string]string{
		"get":    fmt.Sprintf("Retrieves a specific %s connection", connName),
		"list":   fmt.Sprintf("Lists all %s connections", connName),
		"test":   fmt.Sprintf("Tests if a provided %s connection configuration works", connName),
		"upsert": fmt.Sprintf("Creates or updates a %s connection", connName),
		"delete": fmt.Sprintf("Deletes a %s connection", connName),
	}[cmd.Use]
	if short == "" {
		panic(cmd.Use)
	}
	if cmd.Short == "" {
		cmd.Short = short
	}
	argDesc := "[connection-name]"
	if m.defaultArg == "" {
		argDesc = "<connection-name>"
	}
	hasArg := false
	switch m.use {
	case "get", "test", "upsert", "delete":
		cmd.Use += " " + argDesc
		hasArg = true
	}
	if hasArg {
		if m.defaultArg == "" { // there is an arg without default, require 1 arg
			cmd.Args = cobra.ExactArgs(1)
		} else { // there is an arg with default, require 0 or 1 args
			cmd.Args = cobra.MaximumNArgs(1)
			cmd.Long = cmd.Short + fmt.Sprintf(". If connection-name is not provided, %q is assumed as connection name.", m.defaultArg)
		}
	}

	if m.onto != nil {
		flags := NewFlagMapper(cmd, m.onto, up, m.opts)
		cmd.PreRunE = func(cmd *cobra.Command, args []string) error { return flags.MapFlags() }
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		arg := m.defaultArg
		if len(args) > 0 {
			arg = args[0]
		}
		if m.runWithargNoRet != nil {
			return m.runWithargNoRet(arg)
		}
		var resp interface{}
		var err error
		if m.runWithArgRet != nil {
			resp, err = m.runWithArgRet(arg)
		}
		if m.runNoArgRet != nil {
			resp, err = m.runNoArgRet()
		}
		if err != nil {
			return err
		}
		return writeJSON(cmd, resp)
	}

	return cmd
}

func writeJSON(cmd *cobra.Command, v any) error {
	return bite.WriteJSON(cmd.OutOrStdout(), v, true, "")
}

// upload is called for "file ref" arguments. It uploads a file to Lenses and
// returns the corresponding uuid, which gets plugged into the flagMapped
// object.
func upload(fileName string) (uuid.UUID, error) {
	id, err := config.Client.UploadFile(fileName)
	if err != nil {
		return uuid.Nil, err
	}
	golog.Debugf("Uploaded file %q which got assigned uuid %q", fileName, id)
	return id, nil
}

type genericConnectionClient interface {
	GetConnection1(name string) (resp api.ConnectionJsonResponse, err error)
	ListConnections() (resp []api.ConnectionSummaryResponse, err error)
	TestConnection(reqBody api.TestConnectionAPIRequest) (err error)
	UpdateConnection1(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error)
	DeleteConnection1(name string) (err error)
}

// newGenericAPICommand is a helper for connections that are managed via the
// generic API endpoints.
func newGenericAPICommand[T any](templateName string, gen genericConnectionClient, up uploadFunc, opts FlagMapperOpts) []*cobra.Command {
	opts.Descriptions["Tags"] = "Any tags to add to the connection's metadata."
	opts.Descriptions["Update"] = "Set to true if testing an update to an existing connection, false if testing a new connection."
	f := genericAPIAdapter{templateName: templateName, genConnClient: gen}
	testreq := struct {
		O      T
		Update *bool `json:"update,omitempty"`
	}{}
	updreq := struct {
		O    T
		Tags []string `json:"tags"`
	}{}
	return (connCrudsToCobra(templateName, up,
		connCrud{
			use:           "get",
			runWithArgRet: func(arg string) (interface{}, error) { return f.getConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return f.listConnections() },
		}, connCrud{
			use:  "test",
			opts: opts,
			onto: &testreq,
			runWithargNoRet: func(arg string) error {
				return f.testConnection(arg, testreq.Update, testreq.O)
			},
		}, connCrud{
			use:  "upsert",
			opts: opts,
			onto: &updreq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return f.updateConnection(arg, updreq.O, updreq.Tags...)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: f.deleteConnection,
		}))
}

// genericAPIAdapter exposes an API that looks like the "specific" endpoints
// (e.g. Kafka, KafkaConnect, etc.) but uses the generic api under the hood.
type genericAPIAdapter struct {
	templateName  string
	genConnClient genericConnectionClient
}

func (f genericAPIAdapter) getConnection(name string) (resp api.ConnectionJsonResponse, err error) {
	resp, err = f.genConnClient.GetConnection1(name)
	if err != nil {
		return
	}
	// TODO. If the returned template name does not match, do we want to pretend
	// that the connection does not exists?
	return
}

func (f genericAPIAdapter) listConnections() (resp []api.ConnectionSummaryResponse, err error) {
	conns, err := f.genConnClient.ListConnections()
	if err != nil {
		return nil, err
	}
	resp = []api.ConnectionSummaryResponse{} // don't serialise as "null".
	for _, conn := range conns {
		if conn.TemplateName != f.templateName {
			continue
		}
		resp = append(resp, conn)
	}
	return resp, nil
}

func (f genericAPIAdapter) testConnection(name string, update *bool, obj any) (err error) {
	return f.genConnClient.TestConnection(api.TestConnectionAPIRequest{
		Name:                name,
		TemplateName:        f.templateName,
		ConfigurationObject: obj,
		Update:              update,
	})
}

func (f genericAPIAdapter) updateConnection(name string, obj any, tags ...string) (resp api.AddConnectionResponse, err error) {
	return f.genConnClient.UpdateConnection1(name, api.UpsertConnectionAPIRequest{
		ConfigurationObject: obj,
		Tags:                tags,
		TemplateName:        &f.templateName,
	})
}

func (f genericAPIAdapter) deleteConnection(name string) (err error) {
	return f.genConnClient.DeleteConnection1(name)
}
