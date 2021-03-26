package acl

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var validACL = api.ACL{
	PermissionType: api.ACLPermissionType("allow"),
	Principal:      "user:bob",
	Operation:      "read",
	ResourceType:   "topic",
	PatternType:    "literal",
	ResourceName:   "transactions",
	Host:           "acme.com",
}

var invalidACL = api.ACL{
	PermissionType: api.ACLPermissionType("allow"),
	Principal:      "user:bob",
	Operation:      "read",
	ResourceName:   "transactions",
	Host:           "acme.com",
}

func Test_populateACL(t *testing.T) {
	var yamlFiles = []struct {
		path    string
		payload api.ACL
	}{
		{
			path:    "/tmp/validACL.yaml",
			payload: validACL,
		},
		{
			// Missing a couple required params
			path:    "/tmp/invalidACL.yaml",
			payload: invalidACL,
		},
	}

	// Let's create the yaml files to be used for testing and delete them when finished
	for _, file := range yamlFiles {

		out, _ := yaml.Marshal(file.payload)
		err := ioutil.WriteFile(file.path, out, 0644)
		require.NoError(t, err)
		defer os.Remove(file.path)
	}

	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name    string
		args    args
		want    api.ACL
		wantErr bool
	}{
		{
			name: "'acl set' with no arguments or flags should fail",
			args: args{
				cmd:  NewCreateOrUpdateACLCommand(),
				args: []string{},
			},
			want:    api.ACL{},
			wantErr: true,
		},
		{
			name: "'acl set' with valid ACL yaml file",
			args: args{
				cmd:  NewCreateOrUpdateACLCommand(),
				args: []string{"/tmp/validACL.yaml"},
			},
			want:    validACL,
			wantErr: false,
		},
		{
			name: "'acl set' with invalid param",
			args: args{
				cmd:  NewCreateOrUpdateACLCommand(),
				args: []string{"hello.txt"},
			},
			want:    api.ACL{},
			wantErr: true,
		},
		{
			name: "'acl set' with invalid param",
			args: args{
				cmd:  NewCreateOrUpdateACLCommand(),
				args: []string{"hello.yaml"},
			},
			want:    api.ACL{},
			wantErr: true,
		},
		{
			name: "'acl set' with invalid ACL yaml file",
			args: args{
				cmd:  NewCreateOrUpdateACLCommand(),
				args: []string{"/tmp/invalidACL.yaml"},
			},
			want:    invalidACL,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := populateACL(tt.args.cmd, tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("populateACL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("populateACL() = %v, want %v", got, tt.want)
			}
		})
	}

}
