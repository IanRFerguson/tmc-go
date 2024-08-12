/*
Copyright © 2024 IAN R FERGUSON <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/mehanizm/airtable"
	"github.com/spf13/cobra"
)

var memberLibraryCmd = &cobra.Command{
	Use:   "member-library",
	Short: "Interact with the TMC Member Libary",
	Long: `Add domains or confirm that they exist in the Member Library allowlist.

NOTE - We do not support removing domains from the allow list via the command line`,
	Run: func(cmd *cobra.Command, args []string) {
		// Assign values from flags
		_DOMAIN, _ := cmd.Flags().GetString("domain")
		_METHOD, _ := cmd.Flags().GetString("method")
		_DATABASE, _ := cmd.Flags().GetString("database")
		_TABLE, _ := cmd.Flags().GetString("table")

		/* TODO - Make this a function */
		if _DATABASE == "DEFAULT" {
			_DB_ENV, _DB_ENV_EXISTS := os.LookupEnv("AIRTABLE_DATABASE")

			if !_DB_ENV_EXISTS {
				panic("** ERROR ** No --database flag passed in and `AIRTABLE_DATABASE` missing in environment")
			}

			_DATABASE = _DB_ENV
		}

		if _TABLE == "DEFAULT" {
			_TABLE_ENV, _TABLE_ENV_EXISTS := os.LookupEnv("AIRTABLE_TABLE")

			if !_TABLE_ENV_EXISTS {
				panic("** ERROR ** No --database flag passed in and `AIRTABLE_DATABASE` missing in environment")
			}

			_TABLE = _TABLE_ENV

		}

		_API, _API_PRESENT := os.LookupEnv("AIRTABLE_API_KEY")

		if !_API_PRESENT {
			panic("** ERROR ** Missing Airtable API Key - set `AIRTABLE_API_KEY` in your environment")
		}

		// Execute commands
		AIRTABLE_CLIENT := airtable.NewClient(_API)

		if _METHOD == "ADD" {
			addToAirtable(*AIRTABLE_CLIENT, _DOMAIN, _DATABASE, _TABLE)
		} else if _METHOD == "CHECK" {
			checkAirtable(*AIRTABLE_CLIENT, _DOMAIN, _DATABASE, _TABLE)
		} else {
			panic("** ERROR - Invalid method value called")
		}

	},
}

func addToAirtable(airtableClient airtable.Client, domain string, database string, table string) {
	tbl := airtableClient.GetTable(database, table)
	recordsToAdd := airtable.Records{
		Records: []*airtable.Record{
			{
				Fields: map[string]any{
					"Player": domain,
				},
			},
		},
	}

	_, err := tbl.AddRecords(&recordsToAdd)

	if err != nil {
		panic(err)
	}
	fmt.Printf("✅ - %s was successfully added to Airtable\n", domain)
}

func checkAirtable(airtableClient airtable.Client, domain string, database string, table string) {
	tbl := airtableClient.GetTable(database, table)

	records, err := tbl.GetRecords().ReturnFields("Player").InStringFormat("America/New_York", "us").Do()
	var check bool
	check = false
	for _, player := range records.Records {
		playerName := player.Fields["Player"].(string)
		if strings.ToUpper(domain) == strings.ToUpper(playerName) {
			check = true
		}
	}

	if err != nil {
		panic(err)
	}

	if check {
		fmt.Printf("✅ - %s is present in the Airtable database\n", domain)
	} else {
		fmt.Printf("❌ - %s is NOT present in the Airtable database\n", domain)
	}
}

func init() {
	rootCmd.AddCommand(memberLibraryCmd)

	/* TODO - Better type handling here, string "DEFAULT" is bad practice */
	memberLibraryCmd.PersistentFlags().String("domain", "DEFAULT", "Member domain to evaluate")
	memberLibraryCmd.PersistentFlags().String("method", "CHECK", "Should be one of - ADD, CHECK")
	memberLibraryCmd.PersistentFlags().String("database", "DEFAULT", "Airtable Database (forego this by passing in `AIRTABLE_DATABASE` to your environment)")
	memberLibraryCmd.PersistentFlags().String("table", "DEFAULT", "Airtable Table (forego this by passing in `AIRTABLE_TABLE` to your environment)")
}
