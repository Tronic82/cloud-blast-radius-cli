package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/config"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/definitions"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/output"
	"github.com/Tronic82/cloud-blast-radius-cli/internal/parser"
	"github.com/fatih/color"
)

// Shared Globals
var (
	configPath      string
	outputFormat    string
	definitionsFile string
	rulesFile       string
	tfvarsFile      string
	planFile        string
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

type AnalysisResult struct {
	Bindings   []parser.IAMBinding
	Config     *config.Config
	Defs       []parser.ResourceDefinition
	SourceInfo output.SourceInfo
}

func setupAnalysis(args []string) (*AnalysisResult, error) {
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
		return nil, fmt.Errorf("unsupported cloud provider '%s'. Only 'gcp' is currently supported", cfg.CloudProvider)
	}

	// Load Resource Definitions (Embedded or Custom)
	defs, err := definitions.LoadResourceDefinitions(definitionsFile)
	if err != nil {
		return nil, fmt.Errorf("error loading resource definitions: %v", err)
	}

	// Parse Terraform files or plan
	var bindings []parser.IAMBinding
	var sourceInfo output.SourceInfo

	if planFile != "" {
		bindings, err = parser.ParsePlanFile(planFile, defs)
		if err != nil {
			return nil, fmt.Errorf("error parsing plan file: %v", err)
		}
		sourceInfo = output.SourceInfo{Type: "plan_file", Path: planFile, InputMode: "plan_json"}
	} else {
		bindings, err = parser.ParseDir(dir, tfvarsFile, defs, cfg.IgnoredDirectories)
		if err != nil {
			return nil, fmt.Errorf("error parsing directory: %v", err)
		}
		sourceInfo = output.SourceInfo{Type: "directory", Path: dir, InputMode: "hcl"}
	}

	return &AnalysisResult{
		Bindings:   bindings,
		Config:     cfg,
		Defs:       defs,
		SourceInfo: sourceInfo,
	}, nil
}
