package main

import (
	"fmt"
	"sort"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/output"
	"github.com/spf13/cobra"
)

var impactCmd = &cobra.Command{
	Use:   "impact [directory]",
	Short: "Calculate the blast radius",
	Long:  `Analyzes Terraform files to determine the blast radius of IAM principals.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		analysis, err := setupAnalysis(args)
		if err != nil {
			fmt.Printf("Error setting up analysis: %v\n", err)
			return
		}

		results := analyzer.Analyze(analysis.Bindings)

		if outputFormat == "json" {
			jsonOut := output.ConvertToImpactOutput(results, analysis.Config.IsExcluded)
			output.PrintJSON(jsonOut)
			return
		}

		// Text Output
		_, _ = headerColor.Println("\n--- Analysis Results ---")

		principals := make([]string, 0, len(results))
		for p := range results {
			principals = append(principals, p)
		}
		sort.Strings(principals)

		for _, principal := range principals {
			data := results[principal]
			fmt.Printf("\n%s %s\n", principalColor.Sprint("Principal:"), principal)

			// Collect valid resources
			var validResources []string
			for resID, meta := range data.ResourceAccess {
				hasValidRole := false
				for r := range meta.Roles {
					if !analysis.Config.IsExcluded(resID, meta.Type, r) {
						hasValidRole = true
						break
					}
				}
				if hasValidRole {
					validResources = append(validResources, resID)
				}
			}
			sort.Strings(validResources)

			fmt.Printf("  Resources (%d):\n", len(validResources))

			for _, resID := range validResources {
				meta := data.ResourceAccess[resID]
				fmt.Printf("    - %s (%s):\n", resID, meta.Type)

				var validRoles []string
				for r := range meta.Roles {
					if !analysis.Config.IsExcluded(resID, meta.Type, r) {
						validRoles = append(validRoles, r)
					}
				}
				sort.Strings(validRoles)

				for _, r := range validRoles {
					fmt.Printf("      - %s\n", r)
				}
			}
		}
	},
}

func init() {
	impactCmd.Flags().BoolP("visual", "v", false, "Enable visual output (placeholder)")
	impactCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	impactCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")
	rootCmd.AddCommand(impactCmd)
}
