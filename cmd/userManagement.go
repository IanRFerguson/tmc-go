/*
Copyright © 2024 NAME HERE IANFERGUSONRVA@gmail.com
*/
package tmc

import (
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/spf13/cobra"
)

// userManagementCmd represents the userManagement command
var userManagementCmd = &cobra.Command{
	Use:   "add-user",
	Args: cobra.ExactArgs(1),
	Short: "Add a BigQuery user",
	Long: `BIGQUERY USERS CLI
	
Adds a BigQuery user record to two core segmentation tables.

raw_metadata.users
raw_segmentation_metadata.user_data_owners
`,
	Run: func(tmc *cobra.Command, args []string) {
		_USER := args[0]
		_PROJECT, _ := tmc.Flags().GetString("project")
		_TYPE, _ := tmc.Flags().GetString("type")
		_ORG_ID, _  := tmc.Flags().GetString("org-id")
		_MEMBER, _ := tmc.Flags().GetBool("member")
		_AFFILIATE, _ := tmc.Flags().GetBool("affiliate")

		client := setupBigQuery(_PROJECT)
		client.Query("select foo from bar")

		// Validate user input
		isMember := inferOrgType(_MEMBER, _AFFILIATE)
		validateUserInput(client, _USER, _ORG_ID)

		// Get new user ID value
		userID := getUserID(client)

		// Get associated data owner ID value
		dataOwnerID := getDataOwnerID(client, _ORG_ID, isMember)

		// Add user to raw_metadata.users
		addUserRecord(client, _USER, userID, _TYPE, _ORG_ID)

		// Add user to raw_segmentation_metadata.user_data_owners
		addUserDataOwner(client, userID, dataOwnerID)
	},
}

func inferOrgType(member bool, affiliate bool) bool {
	return true
}

func validateUserInput(client *bigquery.Client, user string, orgID string) bool {
	fmt.Println("Validating user input...")

	return true
}

func getUserID(client *bigquery.Client) int {
	return 1
}

func getDataOwnerID(client *bigquery.Client, orgID string, isMember bool) int {
	return 1
}

func addUserRecord(client *bigquery.Client, user string, userID int, userType string, orgID string) {
	fmt.Println("Adding user record...")
}

func addUserDataOwner(client *bigquery.Client, userID int, dataOwnerID int) {
	fmt.Println("Adding user data owner...")
}

func init() {
	rootCmd.AddCommand(userManagementCmd)

	rootCmd.PersistentFlags().String("project", "tmc-dev-394022", "GCP Project ID to write to")
	rootCmd.PersistentFlags().String("org-id", "NONE", "Member or Affiliate ID of affiliated organization")
	rootCmd.PersistentFlags().String("type", "NONE", "Staff, Consultant, Service Account, etc.")
	rootCmd.PersistentFlags().Bool("member", false, "If True, we assume this user belongs to a TMC Member")
	rootCmd.PersistentFlags().Bool("affiliate", false, "If True, we assume this user belongs to a TMC Affiliate")
}
