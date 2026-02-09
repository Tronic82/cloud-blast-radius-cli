package output

import (
	"sort"
	"time"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
)

// NewHierarchyOutput represents the comprehensive JSON output for the hierarchy command
type NewHierarchyOutput struct {
	Version            string                             `json:"version"`
	Provider           string                             `json:"provider"`
	Timestamp          time.Time                          `json:"timestamp"`
	Source             SourceInfo                         `json:"source"`
	Hierarchy          HierarchyInfo                      `json:"hierarchy"`
	HierarchicalAccess []analyzer.HierarchicalAccessEntry `json:"hierarchical_access"`
	Warnings           []analyzer.Warning                 `json:"warnings"`
	Summary            HierarchySummary                   `json:"summary"`
}

// SourceInfo describes where the analysis input came from
type SourceInfo struct {
	Type      string `json:"type"` // "directory" or "plan_file"
	Path      string `json:"path"`
	InputMode string `json:"input_mode"` // "hcl" or "plan_json"
}

// HierarchyInfo contains the discovered hierarchy structure
type HierarchyInfo struct {
	Nodes   []analyzer.HierarchyNode    `json:"nodes"`
	Unknown []analyzer.UnknownHierarchy `json:"unknown"`
}

// HierarchySummary provides summary statistics
type HierarchySummary struct {
	TotalBindingsAnalyzed            int            `json:"total_bindings_analyzed"`
	HierarchicalBindings             int            `json:"hierarchical_bindings"`
	PrincipalsWithHierarchicalAccess int            `json:"principals_with_hierarchical_access"`
	UnknownHierarchyCount            int            `json:"unknown_hierarchy_count"`
	UnknownRolesCount                int            `json:"unknown_roles_count"`
	ByScope                          map[string]int `json:"by_scope"`
	ByAccessType                     map[string]int `json:"by_access_type"`
}

// ConvertToNewHierarchyOutput converts analyzer results to the new comprehensive format
func ConvertToNewHierarchyOutput(result *analyzer.HierarchyAnalysisResult, source SourceInfo, totalBindings int) NewHierarchyOutput {
	// Build summary
	summary := HierarchySummary{
		TotalBindingsAnalyzed: totalBindings,
		ByScope:               make(map[string]int),
		ByAccessType:          make(map[string]int),
	}

	principals := make(map[string]bool)
	for _, entry := range result.HierarchicalAccess {
		principals[entry.Principal] = true
		summary.ByScope[entry.Scope.Type]++
		summary.ByAccessType[entry.Grants.AccessType]++
	}

	summary.PrincipalsWithHierarchicalAccess = len(principals)
	summary.HierarchicalBindings = len(result.HierarchicalAccess)
	summary.UnknownHierarchyCount = len(result.Unknown)

	// Count unknown role warnings
	for _, w := range result.Warnings {
		if w.Type == "unknown_role" {
			summary.UnknownRolesCount++
		}
	}

	// Sort hierarchical access by principal for consistent output
	sortedAccess := make([]analyzer.HierarchicalAccessEntry, len(result.HierarchicalAccess))
	copy(sortedAccess, result.HierarchicalAccess)
	sort.Slice(sortedAccess, func(i, j int) bool {
		if sortedAccess[i].Principal != sortedAccess[j].Principal {
			return sortedAccess[i].Principal < sortedAccess[j].Principal
		}
		if sortedAccess[i].Scope.Type != sortedAccess[j].Scope.Type {
			return sortedAccess[i].Scope.Type < sortedAccess[j].Scope.Type
		}
		return sortedAccess[i].Scope.ID < sortedAccess[j].Scope.ID
	})

	return NewHierarchyOutput{
		Version:   "1.0",
		Provider:  "gcp",
		Timestamp: time.Now().UTC(),
		Source:    source,
		Hierarchy: HierarchyInfo{
			Nodes:   result.Nodes,
			Unknown: result.Unknown,
		},
		HierarchicalAccess: sortedAccess,
		Warnings:           result.Warnings,
		Summary:            summary,
	}
}
