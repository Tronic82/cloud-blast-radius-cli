package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "blast-radius",
	Short: "Blast Radius determines the impact of IAM changes",
	Long:  `A tool to calculate the blast radius of IAM principals in Terraform code.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "blast-radius.yaml", "config file (default is blast-radius.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "Output format: text or json")

	// Custom definitions flags (Optional)
	rootCmd.PersistentFlags().StringVar(&definitionsFile, "definitions", "", "Path to custom resource definitions file")
	rootCmd.PersistentFlags().StringVar(&rulesFile, "rules", "", "Path to custom validation rules file")
}
