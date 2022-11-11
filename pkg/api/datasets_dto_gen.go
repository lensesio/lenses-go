package api

// This code is automatically generated based on the open api spec of:
// Lenses API, version: v5.0.0-12-g2721a0e54.

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// DatasetTag is part of the datasets DTOs.
type DatasetTag struct {
	Name string `json:"name"` // Required.
}

// Lenses is part of the datasets DTOs.
type Lenses struct {
	LensesDataType string `json:"lensesDataType"` // Required.
}

// Native is part of the datasets DTOs.
type Native struct {
	LensesDataType string `json:"lensesDataType"` // Required.
	Native         string `json:"native"`         // Required.
}

// FieldTypeDetails is either one of the following types: Lenses, Native.
type FieldTypeDetails interface{}

// Highlight is part of the datasets DTOs.
type Highlight struct {
	ArrayIndex int    `json:"arrayIndex"` // Required.
	EndIndex   int    `json:"endIndex"`   // Required.
	FieldName  string `json:"fieldName"`  // Required.
	StartIndex int    `json:"startIndex"` // Required.
}

// Field is part of the datasets DTOs.
type Field struct {
	IsNullable  bool             `json:"isNullable"`            // Required.
	Name        string           `json:"name"`                  // Required.
	TypeDetails FieldTypeDetails `json:"typeDetails"`           // Required.
	Ancestors   []string         `json:"ancestors,omitempty"`   // Optional.
	Default     *string          `json:"default,omitempty"`     // Optional.
	Description *string          `json:"description,omitempty"` // Optional.
	Highlights  []Highlight      `json:"highlights,omitempty"`  // Optional.
}

// Fields is part of the datasets DTOs.
type Fields struct {
	Key   []Field `json:"key,omitempty"`   // Optional.
	Value []Field `json:"value,omitempty"` // Optional.
}

// PolicyFieldMatch is part of the datasets DTOs.
type PolicyFieldMatch struct {
	Name    string   `json:"name"`              // Required.
	Parents []string `json:"parents,omitempty"` // Optional.
}

// PolicyMatchDetails is part of the datasets DTOs.
type PolicyMatchDetails struct {
	Obfuscation         string             `json:"obfuscation"`                   // Required.
	PolicyCategory      string             `json:"policyCategory"`                // Required.
	PolicyID            string             `json:"policyId"`                      // Required.
	PolicyName          string             `json:"policyName"`                    // Required.
	MatchingKeyFields   []PolicyFieldMatch `json:"matchingKeyFields,omitempty"`   // Optional.
	MatchingValueFields []PolicyFieldMatch `json:"matchingValueFields,omitempty"` // Optional.
}

// Elastic is part of the datasets DTOs.
type Elastic struct {
	ConnectionName string               `json:"connectionName"`        // Required.
	IsSystemEntity bool                 `json:"isSystemEntity"`        // Required.
	Name           string               `json:"name"`                  // Required.
	Permissions    []string             `json:"permissions"`           // Required.
	Replicas       int                  `json:"replicas"`              // Required.
	Shard          int                  `json:"shard"`                 // Required.
	Description    *string              `json:"description,omitempty"` // Optional.
	Fields         *Fields              `json:"fields,omitempty"`      // Optional.
	Highlights     []Highlight          `json:"highlights,omitempty"`  // Optional.
	Policies       []PolicyMatchDetails `json:"policies,omitempty"`    // Optional.
	Records        *int                 `json:"records,omitempty"`     // Optional.
	SizeBytes      *int                 `json:"sizeBytes,omitempty"`   // Optional.
	Tags           []DatasetTag         `json:"tags,omitempty"`        // Optional.
}

// Kafka is part of the datasets DTOs.
type Kafka struct {
	ConnectionName      string               `json:"connectionName"`           // Required.
	Consumers           int                  `json:"consumers"`                // Required.
	IsCompacted         bool                 `json:"isCompacted"`              // Required.
	IsMarkedForDeletion bool                 `json:"isMarkedForDeletion"`      // Required.
	IsSystemEntity      bool                 `json:"isSystemEntity"`           // Required.
	KeyType             string               `json:"keyType"`                  // Required.
	Name                string               `json:"name"`                     // Required.
	Partitions          int                  `json:"partitions"`               // Required.
	Permissions         []string             `json:"permissions"`              // Required.
	RecordsPerSecond    int                  `json:"recordsPerSecond"`         // Required.
	Replication         int                  `json:"replication"`              // Required.
	ValueType           string               `json:"valueType"`                // Required.
	Description         *string              `json:"description,omitempty"`    // Optional.
	Fields              *Fields              `json:"fields,omitempty"`         // Optional.
	Highlights          []Highlight          `json:"highlights,omitempty"`     // Optional.
	Policies            []PolicyMatchDetails `json:"policies,omitempty"`       // Optional.
	Records             *int                 `json:"records,omitempty"`        // Optional.
	RetentionBytes      *int                 `json:"retentionBytes,omitempty"` // Optional.
	RetentionMs         *int                 `json:"retentionMs,omitempty"`    // Optional.
	SizeBytes           *int                 `json:"sizeBytes,omitempty"`      // Optional.
	Tags                []DatasetTag         `json:"tags,omitempty"`           // Optional.
}

// Postgres is part of the datasets DTOs.
type Postgres struct {
	ConnectionName string               `json:"connectionName"`        // Required.
	IsSystemEntity bool                 `json:"isSystemEntity"`        // Required.
	IsView         bool                 `json:"isView"`                // Required.
	Name           string               `json:"name"`                  // Required.
	Permissions    []string             `json:"permissions"`           // Required.
	Description    *string              `json:"description,omitempty"` // Optional.
	Fields         *Fields              `json:"fields,omitempty"`      // Optional.
	Highlights     []Highlight          `json:"highlights,omitempty"`  // Optional.
	Policies       []PolicyMatchDetails `json:"policies,omitempty"`    // Optional.
	Records        *int                 `json:"records,omitempty"`     // Optional.
	SizeBytes      *int                 `json:"sizeBytes,omitempty"`   // Optional.
	Tags           []DatasetTag         `json:"tags,omitempty"`        // Optional.
}

// SchemaReference is part of the datasets DTOs.
type SchemaReference struct {
	SchemaName  string `json:"schemaName"`  // Required.
	SubjectName string `json:"subjectName"` // Required.
	Version     int    `json:"version"`     // Required.
}

// SchemaVersion is part of the datasets DTOs.
type SchemaVersion struct {
	Format     string            `json:"format"`               // Required.
	ID         int               `json:"id"`                   // Required.
	Schema     string            `json:"schema"`               // Required.
	Version    int               `json:"version"`              // Required.
	References []SchemaReference `json:"references,omitempty"` // Optional.
}

// SchemaRegistrySubject is part of the datasets DTOs.
type SchemaRegistrySubject struct {
	ConnectionName string               `json:"connectionName"`          // Required.
	Format         string               `json:"format"`                  // Required.
	IsSystemEntity bool                 `json:"isSystemEntity"`          // Required.
	Name           string               `json:"name"`                    // Required.
	Permissions    []string             `json:"permissions"`             // Required.
	Schema         string               `json:"schema"`                  // Required.
	SchemaID       int                  `json:"schemaId"`                // Required.
	Version        int                  `json:"version"`                 // Required.
	Compatibility  *string              `json:"compatibility,omitempty"` // Optional.
	Description    *string              `json:"description,omitempty"`   // Optional.
	Fields         *Fields              `json:"fields,omitempty"`        // Optional.
	Highlights     []Highlight          `json:"highlights,omitempty"`    // Optional.
	Policies       []PolicyMatchDetails `json:"policies,omitempty"`      // Optional.
	Records        *int                 `json:"records,omitempty"`       // Optional.
	References     []SchemaReference    `json:"references,omitempty"`    // Optional.
	SizeBytes      *int                 `json:"sizeBytes,omitempty"`     // Optional.
	Tags           []DatasetTag         `json:"tags,omitempty"`          // Optional.
	Versions       []SchemaVersion      `json:"versions,omitempty"`      // Optional.
}

// DatasetMatch either one of the following types: Elastic, Kafka, Postgres,
// SchemaRegistrySubject.
type DatasetMatch interface{}

// PageDatasetMatch is part of the datasets DTOs.
type PageDatasetMatch struct {
	PagesAmount int            `json:"pagesAmount"`      // Required.
	TotalCount  int            `json:"totalCount"`       // Required.
	Values      []DatasetMatch `json:"values,omitempty"` // Optional.
}

// SourceType is part of the datasets DTOs.
type SourceType string

// SourceType enum values.
const (
	SourceTypeKafka                 SourceType = "Kafka"
	SourceTypeElastic               SourceType = "Elastic"
	SourceTypePostgres              SourceType = "Postgres"
	SourceTypeSchemaRegistrySubject SourceType = "SchemaRegistrySubject"
)

// Results is part of the datasets DTOs.
type Results struct {
	Datasets    PageDatasetMatch `json:"datasets"`              // Required.
	SourceTypes []SourceType     `json:"sourceTypes,omitempty"` // Optional.
}

// UnmarshalJSON is a custom unmarshaller to deal with "polymorphic" types.
func (pdm *PageDatasetMatch) UnmarshalJSON(data []byte) error {
	var partial struct {
		PagesAmount int               `json:"pagesAmount"`      // Required.
		TotalCount  int               `json:"totalCount"`       // Required.
		Values      []json.RawMessage `json:"values,omitempty"` // Optional.
	}
	if err := json.Unmarshal(data, &partial); err != nil {
		return err
	}
	pdm.PagesAmount = partial.PagesAmount
	pdm.TotalCount = partial.TotalCount
	polyObj := polyTypeObjUnmarshaller[DatasetMatch, SourceType]{
		discriminatorKey: "sourceType",
		type2ptr: map[SourceType]any{
			SourceTypeElastic:               &Elastic{},
			SourceTypeKafka:                 &Kafka{},
			SourceTypePostgres:              &Postgres{},
			SourceTypeSchemaRegistrySubject: &SchemaRegistrySubject{},
		},
	}
	var err error
	if pdm.Values, err = polyObj.unmarshalSlice(partial.Values); err != nil {
		return fmt.Errorf("unmarshal values: %w", err)
	}
	return nil
}

// polyTypeObjUnmarshaller aims to help with Lenses "polymorphic" JSON object
// types. Users specify the key name of the JSON object that contains the
// type-name of the object, and a mapping from type-name to Go type. It then
// unmarshals json.RawMessages into the correct Go type based on this
// information.
type polyTypeObjUnmarshaller[T any, K ~string] struct {
	discriminatorKey string    // JSON key whose value contains the object's type.
	type2ptr         map[K]any // A map from type name to a pointer to corresponding Go type.
}

// unmarshalSlice calls unmarshal for a slice.
func (t polyTypeObjUnmarshaller[T, K]) unmarshalSlice(ds []json.RawMessage) ([]T, error) {
	os := make([]T, len(ds))
	for i, v := range ds {
		o, err := t.unmarshal(v)
		if err != nil {
			return nil, err
		}
		os[i] = o
	}
	return os, nil
}

// unmarshal
func (t polyTypeObjUnmarshaller[T, K]) unmarshal(d json.RawMessage) (o T, err error) {
	temp := map[string]any{}
	if err := json.Unmarshal(d, &temp); err != nil {
		return o, err
	}
	tn, ok := temp[t.discriminatorKey]
	if !ok {
		return o, fmt.Errorf("json object misses required key: %q", t.discriminatorKey)
	}
	tns, ok := tn.(string)
	if !ok {
		return o, fmt.Errorf("key %q is not of type string but: %T", t.discriminatorKey, tn)
	}
	destTypePtr, ok := t.type2ptr[K(tns)] // A pointer to the correct Go type.
	if !ok {
		return o, fmt.Errorf("no corresponding go type defined for type: %q", tns)
	}
	y := reflect.ValueOf(destTypePtr)
	if y.Kind() != reflect.Ptr {
		return o, fmt.Errorf("%q does not map to a ptr", tns)
	}
	if err := json.Unmarshal(d, destTypePtr); err != nil {
		return o, fmt.Errorf("unmarshal into %T: %w", destTypePtr, err)
	}
	// Let's return the value itself rather than a pointer to it, to be
	// consistent with other DTOs. E.g. slices are []myType rather than
	// []*myType. The only routes I could think of were convoluted generics or
	// reflection. PRs welcome.
	i, ok := y.Elem().Interface().(T)
	if !ok {
		return o, fmt.Errorf("cannot convert %T into %T", y.Elem().Interface(), o)
	}
	return i, nil
}
