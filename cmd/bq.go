/*
Copyright © 2024 NAME HERE IANFERGUSONRVA@gmail.com
*/

package tmc

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/bigquery"
)

func setupBigQuery(projectID string) *bigquery.Client {
	fmt.Println("Setting up BigQuery...")
	setupEnvironment()
	ctx := context.Background()

	// Creates a client.
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("bigquery.NewClient: %v", err)
	}

	return client
}

func setupEnvironment() {
	_, _CREDS_EXIST := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")

	if ! _CREDS_EXIST {
		panic("Missing `GOOGLE_APPLICATION_CREDENTIALS` in runtime environment")
	}
}

// func runQuery(bq *bigquery.Client, query string)