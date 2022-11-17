package api

// This code is automatically generated based on the open api spec of:
// Lenses API, version: v5.0.0-12-g2721a0e54.

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// DatasetField is part of the dataset api query params.
type DatasetField string

// DatasetField enum values.
const (
	DatasetFieldName             DatasetField = "name"
	DatasetFieldRecords          DatasetField = "records"
	DatasetFieldConnectionName   DatasetField = "connectionName"
	DatasetFieldSourceType       DatasetField = "sourceType"
	DatasetFieldIsSystemEntity   DatasetField = "isSystemEntity"
	DatasetFieldRecordsPerSecond DatasetField = "recordsPerSecond"
	DatasetFieldKeyType          DatasetField = "keyType"
	DatasetFieldValueType        DatasetField = "valueType"
	DatasetFieldReplication      DatasetField = "replication"
	DatasetFieldConsumers        DatasetField = "consumers"
	DatasetFieldPartitions       DatasetField = "partitions"
	DatasetFieldRetentionBytes   DatasetField = "retentionBytes"
	DatasetFieldRetentionMs      DatasetField = "retentionMs"
	DatasetFieldSizeBytes        DatasetField = "sizeBytes"
	DatasetFieldReplicas         DatasetField = "replicas"
	DatasetFieldShard            DatasetField = "shard"
	DatasetFieldVersion          DatasetField = "version"
	DatasetFieldFormat           DatasetField = "format"
	DatasetFieldCompatibility    DatasetField = "compatibility"
)

// Order is part of the dataset api query params.
type Order string

// Order enum values.
const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

// RecordCount is part of the dataset api query params.
type RecordCount string

// RecordCount enum values.
const (
	RecordCountEmpty    RecordCount = "empty"
	RecordCountNonEmpty RecordCount = "nonEmpty"
	RecordCountAll      RecordCount = "all"
)

// ListDatasetsParameters contain the query parameters for ListDatasets.
type ListDatasetsParameters struct {
	Page                  *int         // Optional. The page number to be returned, must be greater than zero. Defaults to 1.
	PageSize              int          // Required. The elements amount on a single page, must be greater than zero.
	Query                 *string      // Optional. A search keyword to match dataset, fields and description against.
	Connections           []string     // Optional. A list of connection names to filter by. All connections will be included when no value is supplied.
	Tags                  []string     // Optional. A list of tag names to filter by. All tags will be included when no value is supplied.
	SortBy                DatasetField // Optional. The field to sort results by.
	SortingOrder          Order        // Optional. Sorting order. Defaults to ascending.
	IncludeSystemEntities *bool        // Optional. A flag to include in the search also system entities (e.g. Kafka's `__consumer_offsets` topic).
	IncludeMetadata       *bool        // Optional. Whether to search only by table name, or also to include field names/documentation (defaults to true).
	Format                []string     // Optional. Schema format. Relevant only when sourceType is `ScheamRegistrySubject`.
	RecordCount           *RecordCount // Optional. Controls filter of empty and non-empty topics based on the number of records in it.
}

// ListDatasetsPg hides ListDatasets' paging.
func (c *Client) ListDatasetsPg(params ListDatasetsParameters, maxResults int) (vs []DatasetMatch, err error) {
	for page := 1; ; page++ {
		params.PageSize = 50
		params.Page = &page
		r, err := c.ListDatasets(params)
		if err != nil {
			return nil, err
		}
		vs = append(vs, r.Datasets.Values...)
		if maxResults != 0 && len(vs) >= maxResults { // Be dumb. Over-ask and shrink.
			vs = vs[:maxResults]
			break
		}
		if page >= r.Datasets.PagesAmount {
			break
		}
	}
	return vs, nil
}

// ListDatasets retrieves a list of datasets.
// Tags: Datasets.
func (c *Client) ListDatasets(reqParams ListDatasetsParameters) (res Results, err error) {
	query := url.Values{}
	if reqParams.Page != nil {
		query.Add("page", strconv.Itoa(*reqParams.Page)) // Optional.
	}
	query.Add("pageSize", strconv.Itoa(reqParams.PageSize)) // Required.
	if reqParams.Query != nil {
		query.Add("query", *reqParams.Query) // Optional.
	}
	for _, v := range reqParams.Connections {
		query.Add("connections", v)
	}
	for _, v := range reqParams.Tags {
		query.Add("tags", v)
	}
	if reqParams.SortBy != "" {
		query.Add("sortBy", string(reqParams.SortBy)) // Optional.
	}
	if reqParams.SortingOrder != "" {
		query.Add("sortingOrder", string(reqParams.SortingOrder)) // Optional.
	}
	if reqParams.IncludeSystemEntities != nil {
		query.Add("includeSystemEntities", strconv.FormatBool(*reqParams.IncludeSystemEntities)) // Optional.
	}
	if reqParams.IncludeMetadata != nil {
		query.Add("includeMetadata", strconv.FormatBool(*reqParams.IncludeMetadata)) // Optional.
	}
	for _, v := range reqParams.Format {
		query.Add("format", v)
	}
	if reqParams.RecordCount != nil {
		query.Add("recordCount", string(*reqParams.RecordCount))
	}
	resp, err := c.Do(
		http.MethodGet,
		"/api/v1/datasets?"+query.Encode(),
		contentTypeJSON,
		nil,
	)
	if err != nil {
		return Results{}, fmt.Errorf("client do: %w", err)
	}
	if err := c.ReadJSON(resp, &res); err != nil {
		return Results{}, fmt.Errorf("read json: %w", err)
	}
	return
}
