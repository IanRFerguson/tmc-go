/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package tmc

import (
	"fmt"
	"os"
	"strings"

	"github.com/mehanizm/airtable"
	"github.com/spf13/cobra"
)

var memberLibraryCmd = &cobra.Command{
	Use:   "member-library",
	Args: cobra.ExactArgs(1),
	Short: "Interact with the TMC Member Libary",
	Long: `MEMBER LIBRARY CLI
	
Add domains or confirm that they exist in the Member Library allowlist.

NOTE, we do not support removing domains from the allow list via the command line`,
	Run: func(tmc *cobra.Command, args []string) {
		// Assign values from flags
		_DOMAIN := args[0]

		_FLAGS := tmc.Flags()
		_DATABASE, _ := _FLAGS.GetString("database")
		_TABLE, _ := _FLAGS.GetString("table")
		_ADD_DOMAIN, _ := _FLAGS.GetBool("addDomain")

		var _METHOD string;
		if _ADD_DOMAIN {
			_METHOD = "ADD"
		} else {
			_METHOD = "CHECK"
		}

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


// ABOUT - Writes a new domain value to the Airtable database
func addToAirtable(airtableClient airtable.Client, domain string, database string, table string) bool {
	exists := checkAirtable(airtableClient, domain, database, table)
	if exists {
		fmt.Printf("%s already exists in the Airtable database\n", domain)
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

	fmt.Printf("✅ - %s was written to the Airtable database successfully\n", domain)
	return true
}


// ABOUT - Checks the Airtable database to determine if a value exists
func checkAirtable(airtableClient airtable.Client, domain string, database string, table string) bool {

	check := false
	domainArray := buildDomainArray(airtableClient, database, table)

	// Evaluate if the incoming domain is included in the domain array
	for _, airtableDomain := range domainArray {
		if airtableDomain == domain {
			check = true
		}
	}

	return check
}


// ABOUT - Build an array of domain names that are present in the Airtable database
func buildDomainArray(airtableClient airtable.Client, database string, table string) []string {
	endLoop, offset := false, ""
	var domainArray []string
	
	tbl := airtableClient.GetTable(database, table)
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
			val := rec.Fields["domains"].(string)
			domainArray = append(domainArray, val)
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

	return domainArray
}


// ABOUT - Determine if all the required environment variables are populated
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
	memberLibraryCmd.PersistentFlags().Bool(
		"addDomain", 
		false, 
		"If supplied, the incoming domain will be added to the Airtable Database",
	)
	memberLibraryCmd.PersistentFlags().String(
		"database", 
		"DEFAULT", 
		"Airtable Database (forego this by passing in `AIRTABLE_DATABASE` to your environment)",
	)
	memberLibraryCmd.PersistentFlags().String(
		"table", 
		"DEFAULT", 
		"Airtable Table (forego this by passing in `AIRTABLE_TABLE` to your environment)",
	)
}
