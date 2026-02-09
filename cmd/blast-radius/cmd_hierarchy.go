package main

import (
	"fmt"
	"sort"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/output"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var hierarchyCmd = &cobra.Command{
	Use:   "hierarchy [directory]",
	Short: "Analyze hierarchical access from organization/folder/project-level roles",
	Long:  `Analyzes IAM bindings at organization, folder, and project levels to determine hierarchical access to resources.`,
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

		// Perform hierarchy analysis
		result := analyzer.AnalyzeHierarchy(analysis.Bindings)

		if outputFormat == "json" {
			jsonOut := output.ConvertToNewHierarchyOutput(result, analysis.SourceInfo, len(analysis.Bindings))
			output.PrintJSON(jsonOut)
			return
		}

		// Text output
		_, _ = headerColor.Println("\n--- Hierarchical Access Report ---")

		if len(result.HierarchicalAccess) == 0 {
			fmt.Println("\nNo hierarchical access detected.")
			fmt.Println("(Hierarchical access occurs when roles are granted at organization, folder, or project level)")
			return
		}

		// Group by principal for cleaner output
		byPrincipal := make(map[string][]analyzer.HierarchicalAccessEntry)
		for _, entry := range result.HierarchicalAccess {
			byPrincipal[entry.Principal] = append(byPrincipal[entry.Principal], entry)
		}

		// Sort principals
		principals := make([]string, 0, len(byPrincipal))
		for p := range byPrincipal {
			principals = append(principals, p)
		}
		sort.Strings(principals)

		for _, principal := range principals {
			entries := byPrincipal[principal]
			fmt.Printf("\n%s %s\n", principalColor.Sprint("Principal:"), principal)
			_, _ = headerColor.Println("  Hierarchical Access:")

			for _, entry := range entries {
				displayName := entry.Grants.DisplayName
				if displayName == "" {
					displayName = "resources"
				}
				fmt.Printf("    - %s access to ALL %ss in %s '%s' via role %s assigned on %s level\n",
					colorizeAccessType(entry.Grants.AccessType),
					displayName,
					entry.Scope.Type,
					entry.Scope.ID,
					entry.Role,
					entry.Scope.Type,
				)
			}
		}

		// Print warnings
		if len(result.Warnings) > 0 {
			color.Yellow("\nWarnings:")
			for _, w := range result.Warnings {
				fmt.Printf("  - [%s] %s\n", w.Type, w.Message)
			}
		}

		// Print summary
		fmt.Printf("\n%s %d principals with hierarchical access across %d bindings\n",
			headerColor.Sprint("Summary:"), len(principals), len(result.HierarchicalAccess))
	},
}

func init() {
	hierarchyCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	hierarchyCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")
	rootCmd.AddCommand(hierarchyCmd)
}
