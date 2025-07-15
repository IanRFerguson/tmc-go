/*
Copyright © 2024 IAN FERGUSON IAN@ianferguson.dev
*/

package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

/*
ABOUT: This is a custom object that
stores the instantiated BQ client and
the relevant project ID

Args:
	client			This is an instantiated BigQuery client
	project_id		String representation of the BigQuery project
*/
type BigQueryClient struct {
	client     *bigquery.Client
	project_id string
}

/*
ABOUT: Sets up the BigQuery client
and manages error handling

Args:
	ctx
	project_id
*/
func setupClient(ctx context.Context, project_id string) (*BigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, project_id)

	if err != nil {
		return nil, err
	}

	return &BigQueryClient{client, project_id}, nil
}

/*
ABOUT: Maps onto the `BigQueryClient` object, and
executes a query against the target BigQuery project

Args:
	ctx
	sql
*/
func (bq *BigQueryClient) query(ctx context.Context, sql string) error {
	__query := bq.client.Query(sql)
	resp, err := __query.Read(ctx)

	if err != nil {
		return err
	}

	// Loop through row results and append to a results interface
	var results []map[string]interface{}
	for {
		var row map[string]bigquery.Value

		err := resp.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		rowData := make(map[string]interface{})
		for key, value := range row {
			rowData[key] = value
		}
		results = append(results, rowData)
	}

	// If there are results, print them to the console in a JSON-like format
	if len(results) > 0 {
		output, err := json.MarshalIndent(results, "", "  ")

		if err != nil {
			return err
		}

		fmt.Println(string(output))
	}

	return nil
}
