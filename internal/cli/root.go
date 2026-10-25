/*
Copyright © 2024 IAN R FERGUSON IANFERGUSONRVA@gmail.com
*/
package cli

import (
	"os"

	"github.com/IanRFerguson/tmc-go/internal/asana"
	"github.com/IanRFerguson/tmc-go/internal/memberlibrary"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tmc",
	Short: "tmc is a set of command line tools optimized for The Movement Cooperative",
	Long: `tmc is a CLI library for The Movement Cooperative written in Go.

It's intended to streamline development environments and minimize production code.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(asana.Command, memberlibrary.Command)
}
