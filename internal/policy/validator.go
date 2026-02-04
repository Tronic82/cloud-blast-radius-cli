package policy

import (
	"blast-radius/internal/analyzer"
	"blast-radius/internal/parser"
	"fmt"
)

// PolicyValidator validates IAM configuration against policies
type PolicyValidator struct {
	config         *PolicyConfig
	bindings       []parser.IAMBinding
	directAccess   map[string]*analyzer.PrincipalData
	impGraph       *analyzer.ImpersonationGraph
	canImpersonate func(string, string) bool
}

// NewValidator creates a new policy validator
func NewValidator(
	config *PolicyConfig,
	bindings []parser.IAMBinding,
	directAccess map[string]*analyzer.PrincipalData,
	impGraph *analyzer.ImpersonationGraph,
	canImpersonate func(string, string) bool,
) *PolicyValidator {
	return &PolicyValidator{
		config:         config,
		bindings:       bindings,
		directAccess:   directAccess,
		impGraph:       impGraph,
		canImpersonate: canImpersonate,
	}
}

// Validate runs all policy validations and returns a report
func (v *PolicyValidator) Validate() (*ValidationReport, error) {
	report := &ValidationReport{
		TotalPolicies:     len(v.config.Policies),
		Violations:        []Violation{},
		CompliantPolicies: []string{},
	}

	// Count total principals
	report.PrincipalsAnalyzed = len(v.directAccess)

	// Validate each policy
	for _, policy := range v.config.Policies {
		violations := v.ValidatePolicy(&policy)

		if len(violations) == 0 {
			report.CompliantPolicies = append(report.CompliantPolicies, policy.Name)
		} else {
			report.Violations = append(report.Violations, violations...)
			report.TotalViolations += len(violations)

			// Count by severity
			for _, v := range violations {
				switch v.Severity {
				case SeverityError:
					report.ErrorCount++
				case SeverityWarning:
					report.WarningCount++
				case SeverityInfo:
					report.InfoCount++
				}
			}
		}
	}

	// Calculate max chain depth
	report.MaxChainDepth = v.calculateMaxChainDepth()

	return report, nil
}

// ValidatePolicy validates a single policy
func (v *PolicyValidator) ValidatePolicy(policy *Policy) []Violation {
	switch policy.Type {
	case PolicyTypeRoleRestriction:
		return v.validateRoleRestriction(policy)
	case PolicyTypePersona:
		return v.validatePersona(policy)
	case PolicyTypeResourceAccess:
		return v.validateResourceAccess(policy)
	case PolicyTypeSeparationOfDuty:
		return v.validateSeparationOfDuty(policy)
	case PolicyTypeImpersonationEscalation:
		return v.validateImpersonationEscalation(policy)
	case PolicyTypeEffectiveAccess:
		return v.validateEffectiveAccess(policy)
	default:
		return []Violation{{
			PolicyName:    policy.Name,
			ViolationType: "unknown_type",
			Severity:      SeverityError,
			Message:       "Unknown policy type: " + string(policy.Type),
		}}
	}
}

// validateRoleRestriction validates role restriction policies
func (v *PolicyValidator) validateRoleRestriction(policy *Policy) []Violation {
	violations := []Violation{}
	restriction := policy.RoleRestriction

	if restriction == nil {
		return violations
	}

	// Check each principal
	for principal, data := range v.directAccess {
		// Check if principal matches selector
		if !MatchesPrincipalPattern(principal, restriction.Selector.PrincipalPattern) {
			continue
		}

		// Check all roles for this principal
		for resourceID, meta := range data.ResourceAccess {
			for role := range meta.Roles {
				// Check if role is denied
				if len(restriction.DeniedRoles) > 0 && IsRoleIn(role, restriction.DeniedRoles) {
					violations = append(violations, Violation{
						PolicyName:    policy.Name,
						ViolationType: ViolationTypeForbiddenRole,
						Severity:      policy.Severity,
						Principal:     principal,
						Resource:      resourceID,
						Role:          role,
						Message:       "Principal has forbidden role",
						Remediation:   "Remove role binding or update policy",
					})
				}

				// Check if role is not allowed (if allowed list specified)
				if len(restriction.AllowedRoles) > 0 && !IsRoleIn(role, restriction.AllowedRoles) {
					violations = append(violations, Violation{
						PolicyName:    policy.Name,
						ViolationType: ViolationTypeForbiddenRole,
						Severity:      policy.Severity,
						Principal:     principal,
						Resource:      resourceID,
						Role:          role,
						Message:       "Principal has role not in allowed list",
						Remediation:   "Remove role binding or add role to allowed list",
					})
				}
			}
		}
	}

	return violations
}

// validateResourceAccess validates resource access policies
func (v *PolicyValidator) validateResourceAccess(policy *Policy) []Violation {
	violations := []Violation{}
	access := policy.ResourceAccess

	if access == nil {
		return violations
	}

	// Find all bindings matching the resource selector
	for _, binding := range v.bindings {
		if !MatchesResourcePattern(binding.ResourceID, access.Selector.ResourcePattern) {
			continue
		}
		if access.Selector.ResourceType != "" && access.Selector.ResourceType != "*" {
			if binding.ResourceType != access.Selector.ResourceType {
				continue
			}
		}

		// Check each member
		for _, member := range binding.Members {
			// Check if member is allowed
			if !IsPrincipalIn(member, access.AllowedPrincipals) {
				violations = append(violations, Violation{
					PolicyName:    policy.Name,
					ViolationType: ViolationTypeUnauthorizedPrincipal,
					Severity:      policy.Severity,
					Principal:     member,
					Resource:      binding.ResourceID,
					Role:          binding.Role,
					Message:       "Unauthorized principal has access to resource",
					Remediation:   "Remove principal from resource access",
				})
			}

			// Check role restrictions if specified
			if len(access.AllowedRolesPerPrincipal) > 0 {
				allowedRoles, exists := access.AllowedRolesPerPrincipal[member]
				if exists && !IsRoleIn(binding.Role, allowedRoles) {
					violations = append(violations, Violation{
						PolicyName:    policy.Name,
						ViolationType: ViolationTypeForbiddenRole,
						Severity:      policy.Severity,
						Principal:     member,
						Resource:      binding.ResourceID,
						Role:          binding.Role,
						Message:       "Principal has unauthorized role on resource",
						Remediation:   "Change role to allowed role or remove binding",
					})
				}
			}
		}
	}

	return violations
}

// validateSeparationOfDuty validates separation of duty policies
func (v *PolicyValidator) validateSeparationOfDuty(policy *Policy) []Violation {
	violations := []Violation{}
	sod := policy.SeparationOfDuty

	if sod == nil {
		return violations
	}

	if sod.Scope == "per_principal" {
		// Check each principal doesn't have conflicting roles
		for principal, data := range v.directAccess {
			allRoles := make(map[string]bool)

			// Collect all roles for this principal
			for _, meta := range data.ResourceAccess {
				for role := range meta.Roles {
					allRoles[role] = true
				}
			}

			// Check for conflicts
			for _, conflictSet := range sod.ConflictingRoles {
				hasCount := 0
				conflictingRoles := []string{}

				for _, role := range conflictSet {
					if allRoles[role] {
						hasCount++
						conflictingRoles = append(conflictingRoles, role)
					}
				}

				if hasCount > 1 {
					violations = append(violations, Violation{
						PolicyName:    policy.Name,
						ViolationType: ViolationTypeConflictingRoles,
						Severity:      policy.Severity,
						Principal:     principal,
						Message:       fmt.Sprintf("Principal has conflicting roles: %v", conflictingRoles),
						Remediation:   "Remove one of the conflicting roles",
					})
				}
			}
		}
	}

	return violations
}

// calculateMaxChainDepth calculates the maximum impersonation chain depth
func (v *PolicyValidator) calculateMaxChainDepth() int {
	maxDepth := 0

	// BFS to find maximum depth
	for principal := range v.directAccess {
		visited := make(map[string]bool)
		queue := []struct {
			principal string
			depth     int
		}{{principal, 0}}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			if visited[current.principal] {
				continue
			}
			visited[current.principal] = true

			if current.depth > maxDepth {
				maxDepth = current.depth
			}

			// Add impersonation targets
			if targets, exists := v.impGraph.Graph[current.principal]; exists {
				for _, target := range targets {
					queue = append(queue, struct {
						principal string
						depth     int
					}{target, current.depth + 1})
				}
			}
		}
	}

	return maxDepth
}
