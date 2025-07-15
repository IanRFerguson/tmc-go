/*
Copyright © 2024 IAN FERGUSON IAN@ianferguson.dev
*/
package tmc

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// userManagementCmd represents the userManagement command
var userManagementCmd = &cobra.Command{
	Use:   "add-user",
	Args:  cobra.ExactArgs(1),
	Short: "Add a BigQuery user",
	Long: `BIGQUERY USERS CLI
	
Adds a BigQuery user record to two core segmentation tables.

raw_metadata.users
raw_segmentation_metadata.user_data_owners
`,
	Run: func(tmc *cobra.Command, args []string) {
		userEmail := args[0]
		projectID, _ := tmc.Flags().GetString("project")
		userType, _ := tmc.Flags().GetString("type")
		rawOrganizationID, _ := tmc.Flags().GetString("org-id")
		organizationType, _ := tmc.Flags().GetString("organization-type")

		organizationID, err := strconv.Atoi(rawOrganizationID)

		// Validate user input
		validateUserInput(userEmail, organizationID, organizationType)

		ctx := context.Background()

		client, err := setupClient(ctx, projectID)

		if err != nil {
			panic(err)
		}

		// Get new user ID value
		userID := getUserID(client)

		// Get associated data owner ID value
		dataOwnerID, dataOwnerCode := getDataOwnerMetadata(client, organizationID, organizationType)
		organizationName := getOrganizationName(client, organizationID, organizationType)

		// Get end user confirmation before writing to tables
		userConfirm(userEmail, userID, dataOwnerID, dataOwnerCode, organizationID, organizationName)

		// Add user to raw_metadata.users
		addUserRecord(client, userEmail, userID, userType, organizationID)

		// Add user to raw_segmentation_metadata.user_data_owners
		addUserDataOwner(client, userID, dataOwnerID)

		defer client.client.Close()
	},
}

func validateUserInput(user string, organizationID int, organizationType string) {
	fmt.Println("Validating user input...")
}

func userConfirm(userEmail string, userID int, dataOwnerID int, dataOwnerCode string, organizationID int, organizationName string) {

}

///

func getUserID(client *BigQueryClient) int {
	return 1
}

func getDataOwnerMetadata(client *BigQueryClient, organizationID int, organizationType string) (int, string) {
	return 1, "1"
}

func getOrganizationName(client *BigQueryClient, organizationID int, organizationType string) string {
	return "1"
}

///

func addUserRecord(client *BigQueryClient, user string, userID int, userType string, organizationID int) {
	fmt.Println("Adding user record...")
}

func addUserDataOwner(client *BigQueryClient, userID int, dataOwnerID int) {
	fmt.Println("Adding user data owner...")
}

///

func init() {
	rootCmd.AddCommand(userManagementCmd)

	userManagementCmd.PersistentFlags().String("project", "tmc-dev-394022", "GCP Project ID to write to")
	userManagementCmd.PersistentFlags().String("type", "NONE", "Staff, Consultant, Service Account, etc.")
	userManagementCmd.PersistentFlags().String("org-type", "NONE", "Member or Affiliate")
	userManagementCmd.PersistentFlags().String("org-id", "NONE", "Member or Affiliate ID of affiliated organization")
}
