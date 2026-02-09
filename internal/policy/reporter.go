package policy

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateReport generates a formatted report from violations
func GenerateReport(report *ValidationReport) string {
	var output strings.Builder

	// Header
	output.WriteString("\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	output.WriteString("Policy Validation Report\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Summary stats
	output.WriteString(fmt.Sprintf("Policies Evaluated: %d\n", report.TotalPolicies))
	output.WriteString(fmt.Sprintf("Violations Found: %d\n", report.TotalViolations))

	if report.ErrorCount > 0 {
		output.WriteString(fmt.Sprintf("Errors: %d\n", report.ErrorCount))
	}
	if report.WarningCount > 0 {
		output.WriteString(fmt.Sprintf("Warnings: %d\n", report.WarningCount))
	}
	if len(report.CompliantPolicies) > 0 {
		output.WriteString(fmt.Sprintf("Compliant: %d\n", len(report.CompliantPolicies)))
	}
	output.WriteString("\n")

	// Group violations by policy
	violationsByPolicy := make(map[string][]Violation)
	for _, v := range report.Violations {
		violationsByPolicy[v.PolicyName] = append(violationsByPolicy[v.PolicyName], v)
	}

	// Sort policy names
	policyNames := make([]string, 0, len(violationsByPolicy))
	for name := range violationsByPolicy {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	// Output violations by policy
	for _, policyName := range policyNames {
		violations := violationsByPolicy[policyName]

		for _, v := range violations {
			output.WriteString(FormatViolation(&v))
			output.WriteString("\n")
		}
	}

	// Output compliant policies
	if len(report.CompliantPolicies) > 0 {
		for _, name := range report.CompliantPolicies {
			output.WriteString(fmt.Sprintf("✅ COMPLIANT: %s\n", name))
		}
		output.WriteString("\n")
	}

	// Detailed analysis section (if applicable)
	if report.PrincipalsAnalyzed > 0 {
		output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		output.WriteString("Detailed Analysis\n")
		output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
		output.WriteString(fmt.Sprintf("Total Principals Analyzed: %d\n", report.PrincipalsAnalyzed))
		if report.MaxChainDepth > 0 {
			output.WriteString(fmt.Sprintf("Maximum Impersonation Chain Depth: %d\n", report.MaxChainDepth))
		}

		if len(report.HighRiskFindings) > 0 {
			output.WriteString("\nHigh-Risk Findings:\n")
			for _, finding := range report.HighRiskFindings {
				output.WriteString(fmt.Sprintf("  ⚠️  %s\n", finding))
			}
		}
		output.WriteString("\n")
	}

	// Summary
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	output.WriteString("Summary\n")
	output.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if report.TotalViolations == 0 {
		output.WriteString("Status: PASSED ✅\n")
		output.WriteString("All policies compliant!\n")
	} else {
		output.WriteString("Status: FAILED ❌\n")
		if report.ErrorCount > 0 {
			output.WriteString(fmt.Sprintf("Errors: %d\n", report.ErrorCount))
		}
		if report.WarningCount > 0 {
			output.WriteString(fmt.Sprintf("Warnings: %d\n", report.WarningCount))
		}
		output.WriteString("\nFix the errors above to achieve compliance.\n")
	}

	return output.String()
}

// FormatViolation formats a single violation for display
func FormatViolation(v *Violation) string {
	var output strings.Builder

	// Header
	output.WriteString(fmt.Sprintf("%s: %s\n", strings.ToUpper(string(v.Severity)), v.PolicyName))
	output.WriteString(fmt.Sprintf("   Violation: %s\n", formatViolationType(v.ViolationType)))
	output.WriteString("\n")

	// Details
	if v.Principal != "" {
		output.WriteString(fmt.Sprintf("   Principal: %s\n", v.Principal))
	}
	if v.Resource != "" {
		output.WriteString(fmt.Sprintf("   Resource: %s\n", v.Resource))
	}
	if v.Role != "" {
		output.WriteString(fmt.Sprintf("   Role: %s\n", v.Role))
	}

	// Impersonation chain if present
	if len(v.ImpersonationChain) > 0 {
		output.WriteString("\n   Impersonation Chain:\n")
		output.WriteString(fmt.Sprintf("     %s\n", v.Principal))
		for _, hop := range v.ImpersonationChain {
			output.WriteString(fmt.Sprintf("       → %s\n", hop))
		}
	}

	// Message
	output.WriteString(fmt.Sprintf("\n   %s\n", v.Message))

	// Remediation
	if v.Remediation != "" {
		output.WriteString(fmt.Sprintf("   Remediation: %s\n", v.Remediation))
	}

	// Location
	if v.Location != "" {
		output.WriteString(fmt.Sprintf("   Location: %s\n", v.Location))
	}

	return output.String()
}

// formatViolationType converts violation type to readable string
func formatViolationType(vt ViolationType) string {
	switch vt {
	case ViolationTypeForbiddenRole:
		return "FORBIDDEN ROLE"
	case ViolationTypeMissingRole:
		return "MISSING REQUIRED ROLE"
	case ViolationTypeUnauthorizedPrincipal:
		return "UNAUTHORIZED PRINCIPAL"
	case ViolationTypeConflictingRoles:
		return "CONFLICTING ROLES"
	case ViolationTypeTransitiveRole:
		return "TRANSITIVE ACCESS VIOLATION (Role)"
	case ViolationTypeTransitiveResource:
		return "TRANSITIVE ACCESS VIOLATION (Resource)"
	case ViolationTypeImpersonationDepth:
		return "IMPERSONATION DEPTH EXCEEDED"
	case ViolationTypePrivilegeEscalation:
		return "PRIVILEGE ESCALATION DETECTED"
	case ViolationTypeEffectiveAccess:
		return "UNAUTHORIZED EFFECTIVE ACCESS"
	default:
		return string(vt)
	}
}
