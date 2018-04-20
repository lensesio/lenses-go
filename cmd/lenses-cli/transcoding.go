package main

import (
	"bytes"
	"encoding/json"
)

type Transcoder interface {
	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

// DefaultTranscoder will be able to change via a custom flag, only one will be used per command,
// so a shared variable doesn't hurt, each command runs on each own context by design.
var DefaultTranscoder = defaultTranscoder{}

type defaultTranscoder struct{}

func (_ defaultTranscoder) Encode(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

func (_ defaultTranscoder) EncodeIndent(value interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(value, prefix, indent)
}

func (_ defaultTranscoder) Decode(b []byte, outPtr interface{}) error {
	return json.Unmarshal(b, outPtr)
}

func (defaultTranscoder) Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return json.Indent(dst, src, prefix, indent)
}
