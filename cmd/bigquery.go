/*
Copyright © 2024 IAN FERGUSON IANFERGUSONRVA@gmail.com
*/

package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type BigQueryClient struct {
	client 		*bigquery.Client
	project_id 	string
}

func SetupClient(ctx context.Context, projectId string) (*BigQueryClient, error) {
	client, err := bigquery.NewClient(ctx, projectId)

	if err != nil {
		return nil, err
	}

	return &BigQueryClient{client, projectId}, nil
}

func (bq *BigQueryClient) RunQuery(ctx context.Context, query string) error {
	__query := bq.client.Query(query)
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