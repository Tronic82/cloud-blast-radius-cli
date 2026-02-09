package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/config"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/parser"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/policy"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Color definitions for output
var (
	headerColor       = color.New(color.Bold, color.FgCyan)
	principalColor    = color.New(color.Bold, color.FgWhite)
	accessRead        = color.New(color.FgGreen)
	accessWrite       = color.New(color.FgYellow)
	accessAdmin       = color.New(color.FgRed)
	accessImpersonate = color.New(color.FgCyan)
)

// colorizeAccessType returns a colored string based on access type
func colorizeAccessType(accessType string) string {
	switch accessType {
	case "read":
		return accessRead.Sprint(accessType)
	case "write":
		return accessWrite.Sprint(accessType)
	case "admin":
		return accessAdmin.Sprint(accessType)
	case "impersonate":
		return accessImpersonate.Sprint(accessType)
	default:
		return accessType
	}
}

var configPath string

func main() {
	var rootCmd = &cobra.Command{
		Use:   "blast-radius",
		Short: "Blast Radius determines the impact of IAM changes",
		Long:  `A tool to calculate the blast radius of IAM principals in Terraform code.`,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "blast-radius.yaml", "config file (default is blast-radius.yaml)")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "text", "Output format: text or json")

	// Custom definitions flags (Optional)
	var definitionsFile string
	var rulesFile string
	rootCmd.PersistentFlags().StringVar(&definitionsFile, "definitions", "", "Path to custom resource definitions file")
	rootCmd.PersistentFlags().StringVar(&rulesFile, "rules", "", "Path to custom validation rules file")

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

	var visual bool
	var tfvarsFile string
	var planFile string
	var impactCmd = &cobra.Command{
		Use:   "impact [directory]",
		Short: "Calculate the blast radius",
		Long:  `Analyzes Terraform files to determine the blast radius of IAM principals.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				cfg = &config.Config{}
			}

			if outputFormat == "text" {
				if planFile != "" {
					fmt.Printf("Analyzing plan file: %s\n", planFile)
				} else {
					fmt.Printf("Analyzing directory: %s\n", dir)
				}
			}

			// Validate Provider
			if cfg.CloudProvider == "" {
				cfg.CloudProvider = "gcp"
			}
			if cfg.CloudProvider != "gcp" {
				fmt.Printf("Error: Unsupported cloud provider '%s'. Only 'gcp' is currently supported.\n", cfg.CloudProvider)
				os.Exit(1)
			}

			// Load Resource Definitions (Embedded or Custom)
			defs, err := definitions.LoadResourceDefinitions(definitionsFile)
			if err != nil {
				fmt.Printf("Error loading resource definitions: %v\n", err)
				os.Exit(1)
			}

			// Parse Terraform files or plan
			var bindings []parser.IAMBinding
			if planFile != "" {
				bindings, err = parser.ParsePlanFile(planFile, defs)
				if err != nil {
					fmt.Printf("Error parsing plan file: %v\n", err)
					os.Exit(1)
				}
			} else {
				bindings, err = parser.ParseDir(dir, tfvarsFile, defs, cfg.IgnoredDirectories)
				if err != nil {
					fmt.Printf("Error parsing directory: %v\n", err)
					os.Exit(1)
				}
			}

			results := analyzer.Analyze(bindings)

			if outputFormat == "json" {
				jsonOut := ConvertToImpactOutput(results, cfg.IsExcluded)
				printJSON(jsonOut)
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
						if !cfg.IsExcluded(resID, meta.Type, r) {
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
						if !cfg.IsExcluded(resID, meta.Type, r) {
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
	impactCmd.Flags().BoolVarP(&visual, "visual", "v", false, "Enable visual output (placeholder)")
	impactCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	impactCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")

	var hierarchyCmd = &cobra.Command{
		Use:   "hierarchy [directory]",
		Short: "Analyze hierarchical access from organization/folder/project-level roles",
		Long:  `Analyzes IAM bindings at organization, folder, and project levels to determine hierarchical access to resources.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				cfg = &config.Config{}
			}

			if outputFormat == "text" {
				if planFile != "" {
					fmt.Printf("Analyzing plan file: %s\n", planFile)
				} else {
					fmt.Printf("Analyzing directory: %s\n", dir)
				}
			}

			// Load Resource Definitions (Embedded or Custom)
			defs, err := definitions.LoadResourceDefinitions(definitionsFile)
			if err != nil {
				fmt.Printf("Error loading resource definitions: %v\n", err)
				os.Exit(1)
			}

			// Load Rules (Embedded or Custom)
			if err := definitions.LoadRules(rulesFile); err != nil {
				fmt.Printf("Error loading rules: %v\n", err)
				os.Exit(1)
			}

			var bindings []parser.IAMBinding
			var sourceInfo SourceInfo
			if planFile != "" {
				bindings, err = parser.ParsePlanFile(planFile, defs)
				if err != nil {
					fmt.Printf("Error parsing plan file: %v\n", err)
					os.Exit(1)
				}
				sourceInfo = SourceInfo{Type: "plan_file", Path: planFile, InputMode: "plan_json"}
			} else {
				bindings, err = parser.ParseDir(dir, tfvarsFile, defs, cfg.IgnoredDirectories)
				if err != nil {
					fmt.Printf("Error parsing directory: %v\n", err)
					os.Exit(1)
				}
				sourceInfo = SourceInfo{Type: "directory", Path: dir, InputMode: "hcl"}
			}

			// Perform hierarchy analysis
			result := analyzer.AnalyzeHierarchy(bindings)

			if outputFormat == "json" {
				jsonOut := ConvertToNewHierarchyOutput(result, sourceInfo, len(bindings))
				printJSON(jsonOut)
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
	hierarchyCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	hierarchyCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")

	var accounts []string
	var analyzeCmd = &cobra.Command{
		Use:   "analyze [directory]",
		Short: "Analyze transitive access via impersonation for specific accounts",
		Long:  `Performs deep analysis of IAM access including impersonation chains for specified accounts.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				cfg = &config.Config{}
			}

			accountsToAnalyze := accounts
			if len(accountsToAnalyze) == 0 {
				accountsToAnalyze = cfg.GetAnalysisAccounts()
			}
			if len(accountsToAnalyze) == 0 {
				fmt.Println("No accounts to analyze. Specify --account or update config.")
				return
			}

			if outputFormat == "text" {
				if planFile != "" {
					fmt.Printf("Analyzing plan file: %s\n", planFile)
				} else {
					fmt.Printf("Analyzing directory: %s\n", dir)
				}
			}

			// Load Resource Definitions (Embedded or Custom)
			defs, err := definitions.LoadResourceDefinitions(definitionsFile)
			if err != nil {
				fmt.Printf("Error loading resource definitions: %v\n", err)
				os.Exit(1)
			}

			// Load Rules (Embedded or Custom)
			if err := definitions.LoadRules(rulesFile); err != nil {
				fmt.Printf("Error loading rules: %v\n", err)
				os.Exit(1)
			}

			// Get impersonation function
			canImpersonate := definitions.GetCanImpersonateFunc()

			var bindings []parser.IAMBinding
			if planFile != "" {
				bindings, err = parser.ParsePlanFile(planFile, defs)
				if err != nil {
					fmt.Printf("Error parsing plan file: %v\n", err)
					os.Exit(1)
				}
			} else {
				bindings, err = parser.ParseDir(dir, tfvarsFile, defs, cfg.IgnoredDirectories)
				if err != nil {
					fmt.Printf("Error parsing directory: %v\n", err)
					os.Exit(1)
				}
			}

			directAccess := analyzer.Analyze(bindings)
			impGraph := analyzer.BuildImpersonationGraphWithFunc(bindings, canImpersonate)

			if outputFormat == "text" {
				_, _ = headerColor.Println("\n--- Transitive Access Analysis ---")
			}

			for _, accountEmail := range accountsToAnalyze {
				transitiveAccess := analyzer.AnalyzeTransitiveAccess(accountEmail, directAccess, impGraph)

				if outputFormat == "json" {
					jsonOut := ConvertToAnalyzeOutput(accountEmail, transitiveAccess)
					printJSON(jsonOut)
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
	analyzeCmd.Flags().StringSliceVar(&accounts, "account", nil, "Accounts to analyze (comma-separated emails)")
	analyzeCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	analyzeCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")

	var policyFile string
	var strictMode bool

	var validateCmd = &cobra.Command{
		Use:   "validate [directory]",
		Short: "Validate IAM configuration against policies",
		Long:  `Validates Terraform IAM configuration against custom organizational policies including transitive access.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}

			if policyFile == "" {
				fmt.Println("Error: --policy flag is required")
				os.Exit(1)
			}

			if outputFormat == "text" {
				if planFile != "" {
					fmt.Printf("Validating plan file: %s\n", planFile)
				} else {
					fmt.Printf("Validating directory: %s\n", dir)
				}
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				cfg = &config.Config{}
			}

			// Load Definitions & Rules
			defs, err := definitions.LoadResourceDefinitions(definitionsFile)
			if err != nil {
				fmt.Printf("Error loading definitions: %v\n", err)
				os.Exit(1)
			}
			if err := definitions.LoadRules(rulesFile); err != nil {
				fmt.Printf("Error loading rules: %v\n", err)
				os.Exit(1)
			}

			canImpersonate := definitions.GetCanImpersonateFunc()

			var bindings []parser.IAMBinding
			if planFile != "" {
				bindings, err = parser.ParsePlanFile(planFile, defs)
				if err != nil {
					fmt.Printf("Error parsing plan file: %v\n", err)
					os.Exit(1)
				}
			} else {
				bindings, err = parser.ParseDir(dir, tfvarsFile, defs, cfg.IgnoredDirectories)
				if err != nil {
					fmt.Printf("Error parsing directory: %v\n", err)
					os.Exit(1)
				}
			}

			directAccess := analyzer.Analyze(bindings)
			impGraph := analyzer.BuildImpersonationGraphWithFunc(bindings, canImpersonate)

			policyConfig, err := policy.LoadPolicies(policyFile)
			if err != nil {
				fmt.Printf("Error loading policy file: %v\n", err)
				os.Exit(1)
			}

			validator := policy.NewValidator(policyConfig, bindings, directAccess, impGraph, canImpersonate)
			report, err := validator.Validate()
			if err != nil {
				fmt.Printf("Error during validation: %v\n", err)
				os.Exit(1)
			}

			if outputFormat == "json" {
				jsonOut := ConvertToValidateOutput(report)
				printJSON(jsonOut)
				if report.ErrorCount > 0 {
					if strictMode && report.WarningCount > 0 {
						os.Exit(1)
					}
					os.Exit(1)
				}
				os.Exit(0)
			}

			output := policy.GenerateReport(report)
			fmt.Println(output)

			if report.ErrorCount > 0 {
				if strictMode && report.WarningCount > 0 {
					os.Exit(1)
				}
				os.Exit(1)
			}
			os.Exit(0)
		},
	}
	validateCmd.Flags().StringVar(&policyFile, "policy", "", "Path to policy YAML file")
	validateCmd.Flags().BoolVar(&strictMode, "strict", false, "Treat warnings as errors")
	validateCmd.Flags().StringVar(&tfvarsFile, "tfvars", "", "Path to terraform.tfvars file")
	validateCmd.Flags().StringVar(&planFile, "plan", "", "Path to terraform plan JSON file")

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(impactCmd)
	rootCmd.AddCommand(hierarchyCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(validateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func promptUser(question string, defaultVal string) bool {
	fmt.Print(question)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "" {
			response = strings.ToLower(defaultVal)
		}
		return response == "y" || response == "yes"
	}
	return false
}
