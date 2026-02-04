package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"blast-radius/internal/analyzer"
	"blast-radius/internal/policy"
)

// OutputFormat is the global flag for output format (text or json)
var outputFormat string

// ImpactOutput represents the JSON output for the impact command
type ImpactOutput struct {
	Command    string            `json:"command"`
	Timestamp  time.Time         `json:"timestamp"`
	Principals []PrincipalOutput `json:"principals"`
}

type PrincipalOutput struct {
	Principal string           `json:"principal"`
	Resources []ResourceOutput `json:"resources"`
}

type ResourceOutput struct {
	ResourceID   string   `json:"resource_id"`
	ResourceType string   `json:"resource_type"`
	Roles        []string `json:"roles"`
}

// HierarchyOutput represents the JSON output for the hierarchy command
type HierarchyOutput struct {
	Command            string                     `json:"command"`
	Timestamp          time.Time                  `json:"timestamp"`
	HierarchicalAccess []HierarchicalAccessOutput `json:"hierarchical_access"`
}

type HierarchicalAccessOutput struct {
	Principal     string   `json:"principal"`
	Project       string   `json:"project"`
	ResourceTypes []string `json:"resource_types"`
}

// AnalyzeOutput represents the JSON output for the analyze command
type AnalyzeOutput struct {
	Command            string                     `json:"command"`
	Timestamp          time.Time                  `json:"timestamp"`
	Account            string                     `json:"account"`
	DirectAccess       []ResourceOutput           `json:"direct_access"`
	HierarchicalAccess []HierarchicalAccessOutput `json:"hierarchical_access"`
	TransitiveAccess   []TransitiveAccessOutput   `json:"transitive_access"`
}

type TransitiveAccessOutput struct {
	ResourceID   string   `json:"resource_id"`
	ResourceType string   `json:"resource_type"`
	Roles        []string `json:"roles"`
	ViaChain     []string `json:"via_chain"`
}

// ValidateOutput represents the JSON output for the validate command
type ValidateOutput struct {
	Command    string            `json:"command"`
	Timestamp  time.Time         `json:"timestamp"`
	Status     string            `json:"status"` // "passed" or "failed"
	Violations []ViolationOutput `json:"violations"`
}

type ViolationOutput struct {
	Policy    string `json:"policy"`
	Severity  string `json:"severity"`
	Principal string `json:"principal"`
	Resource  string `json:"resource"`
	Role      string `json:"role"`
	Message   string `json:"message"`
}

// printJSON encodes and prints the given interface as JSON to stdout
func printJSON(v interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
		os.Exit(1)
	}
}

// ConvertToImpactOutput converts analyzer results to ImpactOutput
func ConvertToImpactOutput(results map[string]*analyzer.PrincipalData, isExcluded func(string, string, string) bool) ImpactOutput {
	out := ImpactOutput{
		Command:   "impact",
		Timestamp: time.Now().UTC(),
	}

	// Sort principals
	principals := make([]string, 0, len(results))
	for p := range results {
		principals = append(principals, p)
	}
	sort.Strings(principals)

	for _, p := range principals {
		data := results[p]
		pOut := PrincipalOutput{
			Principal: p,
			Resources: []ResourceOutput{},
		}

		// Sort resources
		resIDs := make([]string, 0, len(data.ResourceAccess))
		for id := range data.ResourceAccess {
			resIDs = append(resIDs, id)
		}
		sort.Strings(resIDs)

		for _, resID := range resIDs {
			meta := data.ResourceAccess[resID]
			roles := []string{}
			for r := range meta.Roles {
				if !isExcluded(resID, meta.Type, r) {
					roles = append(roles, r)
				}
			}
			sort.Strings(roles)

			if len(roles) > 0 {
				pOut.Resources = append(pOut.Resources, ResourceOutput{
					ResourceID:   resID,
					ResourceType: meta.Type,
					Roles:        roles,
				})
			}
		}

		if len(pOut.Resources) > 0 {
			out.Principals = append(out.Principals, pOut)
		}
	}

	return out
}

// ConvertToHierarchyOutput converts analyzer results to HierarchyOutput
func ConvertToHierarchyOutput(results map[string]*analyzer.PrincipalData) HierarchyOutput {
	out := HierarchyOutput{
		Command:   "hierarchy",
		Timestamp: time.Now().UTC(),
	}

	// Sort principals
	principals := make([]string, 0, len(results))
	for p := range results {
		principals = append(principals, p)
	}
	sort.Strings(principals)

	for _, p := range principals {
		data := results[p]
		if len(data.HierarchicalAccess) == 0 {
			continue
		}

		// Sort projects
		projects := make([]string, 0, len(data.HierarchicalAccess))
		for proj := range data.HierarchicalAccess {
			projects = append(projects, proj)
		}
		sort.Strings(projects)

		for _, proj := range projects {
			resTypesMap := data.HierarchicalAccess[proj]
			resTypes := make([]string, 0, len(resTypesMap))
			for rt := range resTypesMap {
				resTypes = append(resTypes, rt)
			}
			sort.Strings(resTypes)

			out.HierarchicalAccess = append(out.HierarchicalAccess, HierarchicalAccessOutput{
				Principal:     p,
				Project:       proj,
				ResourceTypes: resTypes,
			})
		}
	}

	return out
}

// ConvertToAnalyzeOutput converts transitive access result to AnalyzeOutput
func ConvertToAnalyzeOutput(account string, access *analyzer.TransitiveAccess) AnalyzeOutput {
	out := AnalyzeOutput{
		Command:   "analyze",
		Timestamp: time.Now().UTC(),
		Account:   account,
	}

	if access == nil {
		return out
	}

	// Direct Access
	if access.DirectAccess != nil {
		resIDs := make([]string, 0, len(access.DirectAccess.ResourceAccess))
		for id := range access.DirectAccess.ResourceAccess {
			resIDs = append(resIDs, id)
		}
		sort.Strings(resIDs)

		for _, resID := range resIDs {
			meta := access.DirectAccess.ResourceAccess[resID]
			roles := []string{}
			for r := range meta.Roles {
				roles = append(roles, r)
			}
			sort.Strings(roles)

			out.DirectAccess = append(out.DirectAccess, ResourceOutput{
				ResourceID:   resID,
				ResourceType: meta.Type,
				Roles:        roles,
			})
		}

		// Hierarchical Access
		projects := make([]string, 0, len(access.DirectAccess.HierarchicalAccess))
		for proj := range access.DirectAccess.HierarchicalAccess {
			projects = append(projects, proj)
		}
		sort.Strings(projects)

		for _, proj := range projects {
			resTypesMap := access.DirectAccess.HierarchicalAccess[proj]
			resTypes := make([]string, 0, len(resTypesMap))
			for rt := range resTypesMap {
				resTypes = append(resTypes, rt)
			}
			sort.Strings(resTypes)

			out.HierarchicalAccess = append(out.HierarchicalAccess, HierarchicalAccessOutput{
				Principal:     access.Principal,
				Project:       proj,
				ResourceTypes: resTypes,
			})
		}
	}

	// Transitive Access
	if access.TransitiveAccess != nil {
		transResIDs := make([]string, 0, len(access.TransitiveAccess))
		for id := range access.TransitiveAccess {
			transResIDs = append(transResIDs, id)
		}
		sort.Strings(transResIDs)

		for _, resID := range transResIDs {
			details := access.TransitiveAccess[resID]
			roles := []string{}
			for r := range details.Resource.Roles {
				roles = append(roles, r)
			}
			sort.Strings(roles)

			out.TransitiveAccess = append(out.TransitiveAccess, TransitiveAccessOutput{
				ResourceID:   resID,
				ResourceType: details.Resource.Type,
				Roles:        roles,
				ViaChain:     details.ViaChain,
			})
		}
	}

	return out
}

// ConvertToValidateOutput converts validation report to ValidateOutput
func ConvertToValidateOutput(report *policy.ValidationReport) ValidateOutput {
	out := ValidateOutput{
		Command:    "validate",
		Timestamp:  time.Now().UTC(),
		Status:     "passed",
		Violations: []ViolationOutput{},
	}

	if report.ErrorCount > 0 {
		out.Status = "failed"
	}

	// Convert all violations (errors, warnings, info)
	for _, v := range report.Violations {
		out.Violations = append(out.Violations, ViolationOutput{
			Policy:    v.PolicyName,
			Severity:  string(v.Severity),
			Principal: v.Principal,
			Resource:  v.Resource,
			Role:      v.Role,
			Message:   v.Message,
		})
	}

	return out
}
