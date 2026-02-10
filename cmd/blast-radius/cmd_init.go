package main

import (
	"fmt"
	"os"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Blast Radius project",
	Long:  `Creates a default configuration file in the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if file exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Configuration file '%s' already exists.\n", configPath)
			if promptUser("Use existing file? [Y/n]: ", "y") {
				fmt.Println("Using existing configuration.")
				return
			}
			if !promptUser("Overwrite existing file? [y/N]: ", "n") {
				fmt.Println("Operation aborted.")
				return
			}
		}

		// Prompt for Cloud Provider
		fmt.Println("Select Cloud Provider:")
		fmt.Println("  1. GCP (Google Cloud Platform) [Default]")

		provider := "gcp"
		if !promptUser("Confirm usage of GCP? [Y/n]: ", "y") {
			fmt.Println("Only GCP is supported in this version.")
		}

		if err := config.CreateDefault(configPath, provider); err != nil {
			fmt.Printf("Error creating configuration: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created '%s' with provider: %s\n", configPath, provider)
		fmt.Println("Ready to use! Try running 'blast-radius impact'")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
