package connection

import (
	"testing"

	"github.com/google/uuid"
	cobra "github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type nested struct {
	AnOptInt    *int `json:",omitempty"`
	Bool        bool
	OptBool     *bool `json:",omitempty"`
	GetsDefault string
}

type omgStruct struct {
	Nest           nested
	OneString      string
	OptString      *string `json:",omitempty"`
	MultiString    []string
	OptMultiString []string `json:",omitempty"`
	AnInt          int
	KeyValues1     map[string]string
	FileID         struct {
		FileId uuid.UUID `json:"fileId"`
	}
	OptFileID *struct {
		FileId uuid.UUID `json:"fileId"`
	} `json:",omitempty"`
	Hidden string
}

func ptrTo[T any](v T) *T {
	return &v
}

// TestFlagMapperSmoke lazily tests all bells and whistles in one go.
func TestFlagMapperSmoke(t *testing.T) {
	u0 := uuid.MustParse("64fb0f9e-a6ed-4a4b-8167-bf579fb0d138")
	u1 := uuid.MustParse("0c85c282-942d-45e1-825b-b40fc23a51f5")
	cmd := &cobra.Command{
		Use: "test",
	}
	var dest omgStruct
	m := NewFlagMapper(cmd, &dest, func(s string) (uuid.UUID, error) {
		assert.Equal(t, "my-file", s)
		return u0, nil
	}, FlagMapperOpts{
		Defaults:     map[string]string{"GetsDefault": "good"},
		Descriptions: map[string]string{"AnInt": "sweet"},
		Hide:         []string{"Hidden"},
		Rename:       map[string]string{"KeyValues1": "key-values"},
	})
	assert.Contains(t, cmd.Flags().Lookup("an-int").Usage, "sweet")
	assert.Nil(t, cmd.Flags().Lookup("hidden"))
	cmd.Run = func(cmd *cobra.Command, args []string) {
		err := m.MapFlags()
		require.NoError(t, err)

		assert.Equal(t, omgStruct{
			OneString:   "one",
			OptString:   ptrTo("two"),
			MultiString: []string{"item-1", "item-2"},
			AnInt:       42,
			KeyValues1:  map[string]string{"a": "b", "c": "d"},
			FileID: struct {
				FileId uuid.UUID "json:\"fileId\""
			}{u0},
			OptFileID: &struct {
				FileId uuid.UUID `json:"fileId"`
			}{u1},
			Nest: nested{
				AnOptInt:    ptrTo(31337),
				Bool:        true,
				OptBool:     ptrTo(false),
				GetsDefault: "good",
			},
		}, dest)
	}
	cmd.SetArgs([]string{"test",
		"--one-string", "one",
		"--opt-string", "two",
		"--multi-string", "item-1",
		"--multi-string", "item-2",
		"--an-int", "42",
		"--an-opt-int", "31337",
		"--bool",
		"--opt-bool=false",
		"--key-values", "a=b",
		"--key-values", "c=d",
		"--file-id", "@my-file",
		"--opt-file-id", u1.String()})
	cmd.Execute()

}
