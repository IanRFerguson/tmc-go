/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var memberLibraryCmd = &cobra.Command{
	Use:   "member-library",
	Short: "Interact with the TMC Member Libary",
	Long: `Add domains or confirm that they exist in the Member Library allowlist.

NOTE - We do not support removing domains from the allow list via the command line`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("memberLibrary called")
	},
}

func init() {
	rootCmd.AddCommand(memberLibraryCmd)

	// Command line args
	// memberLibraryCmd
}
