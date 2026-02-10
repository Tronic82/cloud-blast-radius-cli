package main

import (
	"fmt"
	"os"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/output"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/policy"
	"github.com/spf13/cobra"
)

var (
	policyFile string
	strictMode bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [directory]",
	Short: "Validate IAM configuration against policies",
	Long:  `Validates Terraform IAM configuration against custom organizational policies including transitive access.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if policyFile == "" {
			fmt.Println("Error: --policy flag is required")
			os.Exit(1)
		}

		// Load Rules (Embedded or Custom)
		if err := definitions.LoadRules(rulesFile); err != nil {
			fmt.Printf("Error loading rules: %v\n", err)
			return
		}

		analysis, err := setupAnalysis(args)
		if err != nil {
			fmt.Printf("Error setting up analysis: %v\n", err)
			return
		}

		if outputFormat == "text" {
			if planFile != "" {
				fmt.Printf("Validating plan file: %s\n", planFile)
			}
		}

		canImpersonate := definitions.GetCanImpersonateFunc()

		directAccess := analyzer.Analyze(analysis.Bindings)
		impGraph := analyzer.BuildImpersonationGraphWithFunc(analysis.Bindings, canImpersonate)

		policyConfig, err := policy.LoadPolicies(policyFile)
		if err != nil {
			fmt.Printf("Error loading policy file: %v\n", err)
			os.Exit(1)
		}

		validator := policy.NewValidator(policyConfig, analysis.Bindings, directAccess, impGraph, canImpersonate)
		report, err := validator.Validate()
		if err != nil {
			fmt.Printf("Error during validation: %v\n", err)
			os.Exit(1)
		}

		if outputFormat == "json" {
			jsonOut := output.ConvertToValidateOutput(report)
			output.PrintJSON(jsonOut)
			if report.ErrorCount > 0 {
				if strictMode && report.WarningCount > 0 {
					os.Exit(1)
				}
				os.Exit(1)
			}
			os.Exit(0)
		}

		outputStr := policy.GenerateReport(report)
		fmt.Println(outputStr)

		if report.ErrorCount > 0 {
			if strictMode && report.WarningCount > 0 {
				os.Exit(1)
			}
			os.Exit(1)
		}
		os.Exit(0)
	},
}

func init() {
	validateCmd.Flags().StringVar(&policyFile, "policy", "", "Path to policy YAML file")
	validateCmd.Flags().BoolVar(&strictMode, "strict", false, "Treat warnings as errors")
	validateCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	validateCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")
	rootCmd.AddCommand(validateCmd)
}
