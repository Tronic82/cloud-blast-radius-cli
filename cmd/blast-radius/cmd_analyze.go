package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/output"
	"github.com/spf13/cobra"
)

var accounts []string

var analyzeCmd = &cobra.Command{
	Use:   "analyze [directory]",
	Short: "Analyze transitive access via impersonation for specific accounts",
	Long:  `Performs deep analysis of IAM access including impersonation chains for specified accounts.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		accountsToAnalyze := accounts
		if len(accountsToAnalyze) == 0 {
			accountsToAnalyze = analysis.Config.GetAnalysisAccounts()
		}
		if len(accountsToAnalyze) == 0 {
			fmt.Println("No accounts to analyze. Specify --account or update config.")
			return
		}

		// Get impersonation function
		canImpersonate := definitions.GetCanImpersonateFunc()

		directAccess := analyzer.Analyze(analysis.Bindings)
		impGraph := analyzer.BuildImpersonationGraphWithFunc(analysis.Bindings, canImpersonate)

		if outputFormat == "text" {
			_, _ = headerColor.Println("\n--- Transitive Access Analysis ---")
		}

		for _, accountEmail := range accountsToAnalyze {
			transitiveAccess := analyzer.AnalyzeTransitiveAccess(accountEmail, directAccess, impGraph)

			if outputFormat == "json" {
				jsonOut := output.ConvertToAnalyzeOutput(accountEmail, transitiveAccess)
				output.PrintJSON(jsonOut)
				continue
			}

			_, _ = headerColor.Printf("\n=== Analyzing: %s ===\n", accountEmail)
			if transitiveAccess == nil {
				fmt.Printf("  No matching principal found for account: %s\n", accountEmail)
				continue
			}

			fmt.Printf("\n%s %s\n", principalColor.Sprint("Principal:"), transitiveAccess.Principal)

			// Direct Access
			if len(transitiveAccess.DirectAccess.ResourceAccess) > 0 {
				_, _ = headerColor.Println("\nDirect Access:")
				var resources []string
				for resID := range transitiveAccess.DirectAccess.ResourceAccess {
					resources = append(resources, resID)
				}
				sort.Strings(resources)

				for _, resID := range resources {
					meta := transitiveAccess.DirectAccess.ResourceAccess[resID]
					fmt.Printf("  - %s (%s):\n", resID, meta.Type)

					var roles []string
					for role := range meta.Roles {
						roles = append(roles, role)
					}
					sort.Strings(roles)

					for _, role := range roles {
						fmt.Printf("      %s\n", role)
					}
				}
			} else {
				fmt.Printf("\n%s None\n", headerColor.Sprint("Direct Access:"))
			}

			// Hierarchical Access
			if len(transitiveAccess.DirectAccess.HierarchicalAccess) > 0 {
				_, _ = headerColor.Println("\nHierarchical Access:")
				var projects []string
				for proj := range transitiveAccess.DirectAccess.HierarchicalAccess {
					projects = append(projects, proj)
				}
				sort.Strings(projects)
				for _, proj := range projects {
					fmt.Printf("  - All resources in project '%s'\n", proj)
				}
			}

			// Effective Grants (via impersonation)
			if len(transitiveAccess.TransitiveAccess) > 0 {
				_, _ = headerColor.Println("\nEffective Grants (via impersonation):")
				var resources []string
				for resID := range transitiveAccess.TransitiveAccess {
					resources = append(resources, resID)
				}
				sort.Strings(resources)

				for _, resID := range resources {
					accessVia := transitiveAccess.TransitiveAccess[resID]
					fmt.Printf("  - %s (%s):\n", resID, accessVia.Resource.Type)

					var roles []string
					for role := range accessVia.Resource.Roles {
						roles = append(roles, role)
					}
					sort.Strings(roles)

					for _, role := range roles {
						fmt.Printf("      %s %s\n", accessImpersonate.Sprint("[EFFECTIVE]"), role)
					}
					fmt.Printf("    → via chain: %s\n", strings.Join(accessVia.ViaChain, " → "))
				}
			} else {
				fmt.Printf("\n%s None\n", headerColor.Sprint("Effective Grants (via impersonation):"))
			}
		}
	},
}

func init() {
	analyzeCmd.Flags().StringSliceVar(&accounts, "account", nil, "Accounts to analyze (comma-separated emails)")
	analyzeCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	analyzeCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")
	rootCmd.AddCommand(analyzeCmd)
}
