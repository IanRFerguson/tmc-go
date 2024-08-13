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
		_FLAGS := cmd.Flags()
		_DOMAIN, _ := _FLAGS.GetString("domain")
		_METHOD, _ := _FLAGS.GetString("method")
		_DATABASE, _ := _FLAGS.GetString("database")
		_TABLE, _ := _FLAGS.GetString("table")

		_DATABASE = checkEnvironment(_DATABASE, "AIRTABLE_DATABASE", "DEFAULT")
		_TABLE = checkEnvironment(_TABLE, "AIRTABLE_TABLE", "DEFAULT")
		_API := checkEnvironment("", "AIRTABLE_API_KEY", "") // NOTE - This is a little hacky but follows the pattern

		AIRTABLE_CLIENT := airtable.NewClient(_API)
		switch strings.ToUpper(_METHOD) {
		case "ADD":
			addToAirtable(*AIRTABLE_CLIENT, _DOMAIN, _DATABASE, _TABLE)
		case "CHECK":
			exists := checkAirtable(*AIRTABLE_CLIENT, _DOMAIN, _DATABASE, _TABLE)
			if exists {
				fmt.Printf("✅ - %s exists in the Airtable database\n", _DOMAIN)
			} else {
				fmt.Printf("❌ - A record for %s does NOT exist in the Airtable database\n", _DOMAIN)
			}
		default:
			panic("** ERROR - Invalid method value called")
		}

	},
}

func addToAirtable(airtableClient airtable.Client, domain string, database string, table string) bool {
	exists := checkAirtable(airtableClient, domain, database, table)
	if exists {
		fmt.Printf("%s already exists in the Airtable database", domain)
		return true
	}

	tbl := airtableClient.GetTable(database, table)
	recordsToAdd := airtable.Records{
		Records: []*airtable.Record{
			{
				Fields: map[string]any{
					"domains": domain,
				},
			},
		},
	}

	_, err := tbl.AddRecords(&recordsToAdd)
	if err != nil {
		panic(err)
	}
	return true
}

func checkAirtable(airtableClient airtable.Client, domain string, database string, table string) bool {
	tbl := airtableClient.GetTable(database, table)

	/*
		The logic here is as follows...
			* The Airtable API is called
			* The results of the reponse are pushed to an array
			* If an offset is present, it is fed into the next API call
			* If an offset is NOT present, break out of the loop
	*/

	endLoop, check, offset := false, false, ""
	var domainArray []string
	for !endLoop {
		// Hit Airtable API with offset
		records, err := tbl.GetRecords().
			ReturnFields("domains").
			WithOffset(offset).
			Do()

		// Raise errors inline (all or nothing)
		if err != nil {
			panic(err)
		}

		// Iteratively add domain names to the running domain
		for _, rec := range records.Records {
			domainArray = append(domainArray, rec.Fields["domains"].(string))
		}

		/*
			Evaluate the offset value here ... if it's included in the body
			of the API response we can pass it into the next API call
		*/
		offset = records.Offset
		if offset == "" {
			break
		}
	}

	// Evaluate if the incoming domain is included in the domain array
	for _, airtableDomain := range domainArray {
		if airtableDomain == domain {
			check = true
		}
	}

	return check
}

func checkEnvironment(flagValue string, envVariable string, defaultValue string) string {
	if flagValue == defaultValue {
		_ENV, _ENV_EXISTS := os.LookupEnv(envVariable)

		if !_ENV_EXISTS {
			msg := fmt.Sprintf("** ERROR ** No --database flag passed in and `%s` missing in environment", envVariable)
			panic(msg)
		}

		return _ENV
	}

	return flagValue
}

func init() {
	rootCmd.AddCommand(memberLibraryCmd)

	/* TODO - Better type handling here, string "DEFAULT" is bad practice */
	memberLibraryCmd.PersistentFlags().String("domain", "DEFAULT", "Member domain to evaluate")
	memberLibraryCmd.PersistentFlags().String("method", "CHECK", "Should be one of - ADD, CHECK")
	memberLibraryCmd.PersistentFlags().String("database", "DEFAULT", "Airtable Database (forego this by passing in `AIRTABLE_DATABASE` to your environment)")
	memberLibraryCmd.PersistentFlags().String("table", "DEFAULT", "Airtable Table (forego this by passing in `AIRTABLE_TABLE` to your environment)")
}
