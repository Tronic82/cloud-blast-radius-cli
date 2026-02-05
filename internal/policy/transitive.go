package policy

import (
	"fmt"
	"strings"

	"github.com/Tronic82/cloud-blast-radius-cli/internal/analyzer"
)

// validatePersona validates persona policies (including transitive access)
func (v *PolicyValidator) validatePersona(policy *Policy) []Violation {
	violations := []Violation{}
	persona := policy.Persona

	if persona == nil {
		return violations
	}

	for _, principal := range persona.Principals {
		// Validate required bindings
		reqViolations := v.checkRequiredBindings(principal, persona)
		violations = append(violations, reqViolations...)

		// Validate forbidden bindings (direct)
		forbViolations := v.checkForbiddenBindings(principal, persona)
		violations = append(violations, forbViolations...)

		// Validate transitive access if enabled
		if persona.ValidateTransitiveAccess && persona.TransitiveConstraints != nil {
			transViolations := v.validateTransitiveAccess(principal, persona)
			for i := range transViolations {
				transViolations[i].PolicyName = policy.Name
				transViolations[i].Severity = policy.Severity
			}
			violations = append(violations, transViolations...)
		}
	}

	return violations
}

// checkRequiredBindings verifies all required bindings exist
func (v *PolicyValidator) checkRequiredBindings(principal string, persona *PersonaPolicy) []Violation {
	violations := []Violation{}

	// Get principal data
	data, exists := v.directAccess[principal]
	if !exists {
		// Principal doesn't exist in configuration
		for _, required := range persona.RequiredBindings {
			violations = append(violations, Violation{
				ViolationType: ViolationTypeMissingRole,
				Principal:     principal,
				Resource:      required.ResourcePattern,
				Role:          required.Role,
				Message:       fmt.Sprintf("Required binding missing: %s on %s", required.Role, required.ResourcePattern),
				Remediation:   "Add required role binding",
			})
		}
		return violations
	}

	// Check each required binding
	for _, required := range persona.RequiredBindings {
		found := false

		for resourceID, meta := range data.ResourceAccess {
			if !MatchesResourcePattern(resourceID, required.ResourcePattern) {
				continue
			}
			if required.ResourceType != "" && meta.Type != required.ResourceType {
				continue
			}
			if meta.Roles[required.Role] {
				found = true
				break
			}
		}

		if !found {
			violations = append(violations, Violation{
				ViolationType: ViolationTypeMissingRole,
				Principal:     principal,
				Resource:      required.ResourcePattern,
				Role:          required.Role,
				Message:       fmt.Sprintf("Required binding missing: %s on %s", required.Role, required.ResourcePattern),
				Remediation:   "Add required role binding",
			})
		}
	}

	return violations
}

// checkForbiddenBindings verifies no forbidden bindings exist
func (v *PolicyValidator) checkForbiddenBindings(principal string, persona *PersonaPolicy) []Violation {
	violations := []Violation{}

	data, exists := v.directAccess[principal]
	if !exists {
		return violations
	}

	// Check each forbidden binding
	for _, forbidden := range persona.ForbiddenBindings {
		for resourceID, meta := range data.ResourceAccess {
			if !MatchesResourcePattern(resourceID, forbidden.ResourcePattern) {
				continue
			}
			if forbidden.ResourceType != "" && forbidden.ResourceType != "*" && meta.Type != forbidden.ResourceType {
				continue
			}

			// Check if role matches
			if forbidden.Role == "*" {
				// Any role is forbidden
				for role := range meta.Roles {
					violations = append(violations, Violation{
						ViolationType: ViolationTypeForbiddenRole,
						Principal:     principal,
						Resource:      resourceID,
						Role:          role,
						Message:       fmt.Sprintf("Forbidden access to resource: %s", resourceID),
						Remediation:   "Remove role binding",
					})
				}
			} else if meta.Roles[forbidden.Role] {
				violations = append(violations, Violation{
					ViolationType: ViolationTypeForbiddenRole,
					Principal:     principal,
					Resource:      resourceID,
					Role:          forbidden.Role,
					Message:       fmt.Sprintf("Forbidden role %s on resource %s", forbidden.Role, resourceID),
					Remediation:   "Remove role binding",
				})
			}
		}
	}

	return violations
}

// validateTransitiveAccess validates transitive access constraints
func (v *PolicyValidator) validateTransitiveAccess(principal string, persona *PersonaPolicy) []Violation {
	violations := []Violation{}
	constraints := persona.TransitiveConstraints

	// Calculate transitive access for this principal
	email := ExtractPrincipalEmail(principal)
	transitiveAccess := analyzer.AnalyzeTransitiveAccess(email, v.directAccess, v.impGraph)

	if transitiveAccess == nil {
		return violations
	}

	// Check impersonation depth
	maxDepth := v.calculateTransitiveDepth(transitiveAccess)
	if maxDepth > constraints.MaxImpersonationDepth {
		violations = append(violations, Violation{
			ViolationType: ViolationTypeImpersonationDepth,
			Principal:     principal,
			Message:       fmt.Sprintf("Max impersonation depth exceeded: %d (allowed: %d)", maxDepth, constraints.MaxImpersonationDepth),
			Remediation:   "Remove impersonation permissions or increase allowed depth",
		})
	}

	// Check forbidden transitive roles
	for resourceID, accessVia := range transitiveAccess.TransitiveAccess {
		for role := range accessVia.Resource.Roles {
			if IsRoleIn(role, constraints.ForbiddenTransitiveRoles) {
				violations = append(violations, Violation{
					ViolationType:      ViolationTypeTransitiveRole,
					Principal:          principal,
					Resource:           resourceID,
					Role:               role,
					ImpersonationChain: accessVia.ViaChain,
					Message:            fmt.Sprintf("Forbidden transitive role '%s' via impersonation: %s", role, strings.Join(accessVia.ViaChain, " → ")),
					Remediation:        "Remove impersonation permission in chain",
				})
			}
		}

		// Check forbidden transitive resources
		for _, forbiddenResource := range constraints.ForbiddenTransitiveResources {
			if MatchesResourcePattern(resourceID, forbiddenResource.ResourcePattern) {
				if forbiddenResource.ResourceType == "" || forbiddenResource.ResourceType == accessVia.Resource.Type {
					violations = append(violations, Violation{
						ViolationType:      ViolationTypeTransitiveResource,
						Principal:          principal,
						Resource:           resourceID,
						ImpersonationChain: accessVia.ViaChain,
						Message:            fmt.Sprintf("Forbidden transitive resource access: %s via %s", resourceID, strings.Join(accessVia.ViaChain, " → ")),
						Remediation:        "Remove impersonation permission in chain",
					})
					break
				}
			}
		}
	}

	// Check impersonation targets if whitelist specified
	if len(constraints.AllowedImpersonationTargets) > 0 {
		if targets, exists := v.impGraph.Graph[principal]; exists {
			for _, target := range targets {
				if !IsPrincipalIn(target, constraints.AllowedImpersonationTargets) {
					violations = append(violations, Violation{
						ViolationType:      ViolationTypeTransitiveResource,
						Principal:          principal,
						Resource:           target,
						ImpersonationChain: []string{target},
						Message:            fmt.Sprintf("Impersonation of unauthorized target: %s", target),
						Remediation:        "Remove impersonation permission or add target to allowed list",
					})
				}
			}
		}
	}

	return violations
}

// calculateTransitiveDepth calculates the maximum depth in a transitive access result
func (v *PolicyValidator) calculateTransitiveDepth(transitiveAccess *analyzer.TransitiveAccess) int {
	maxDepth := 0

	for _, accessVia := range transitiveAccess.TransitiveAccess {
		if len(accessVia.ViaChain) > maxDepth {
			maxDepth = len(accessVia.ViaChain)
		}
	}

	return maxDepth
}

// validateImpersonationEscalation validates impersonation escalation policies
func (v *PolicyValidator) validateImpersonationEscalation(policy *Policy) []Violation {
	violations := []Violation{}
	escalation := policy.ImpersonationEscalation

	if escalation == nil {
		return violations
	}

	// For each principal, check if they can escalate privileges
	for principal := range v.directAccess {
		email := ExtractPrincipalEmail(principal)
		transitiveAccess := analyzer.AnalyzeTransitiveAccess(email, v.directAccess, v.impGraph)

		if transitiveAccess == nil {
			continue
		}

		// Get all direct roles
		directRoles := make(map[string]bool)
		for _, meta := range transitiveAccess.DirectAccess.ResourceAccess {
			for role := range meta.Roles {
				directRoles[role] = true
			}
		}

		// Check each escalation rule
		for _, rule := range escalation.ForbiddenEscalations {
			// Check role-based escalation
			if rule.FromRolePattern != "" && rule.ToRolePattern != "" {
				for directRole := range directRoles {
					if MatchesRolePattern(directRole, rule.FromRolePattern) {
						// Check if they can get to forbidden role via impersonation
						for _, accessVia := range transitiveAccess.TransitiveAccess {
							for transitiveRole := range accessVia.Resource.Roles {
								if MatchesRolePattern(transitiveRole, rule.ToRolePattern) {
									violations = append(violations, Violation{
										PolicyName:         policy.Name,
										ViolationType:      ViolationTypePrivilegeEscalation,
										Severity:           policy.Severity,
										Principal:          principal,
										Role:               transitiveRole,
										ImpersonationChain: accessVia.ViaChain,
										Message:            fmt.Sprintf("Privilege escalation detected: %s → %s via impersonation", directRole, transitiveRole),
										Remediation:        "Remove impersonation permission in chain",
									})
								}
							}
						}
					}
				}
			}

			// Check principal/resource-based escalation
			if rule.FromPrincipalPattern != "" && rule.ToResourcePattern != "" {
				if MatchesPrincipalPattern(principal, rule.FromPrincipalPattern) {
					for resourceID := range transitiveAccess.TransitiveAccess {
						if MatchesResourcePattern(resourceID, rule.ToResourcePattern) {
							violations = append(violations, Violation{
								PolicyName:         policy.Name,
								ViolationType:      ViolationTypePrivilegeEscalation,
								Severity:           policy.Severity,
								Principal:          principal,
								Resource:           resourceID,
								ImpersonationChain: transitiveAccess.TransitiveAccess[resourceID].ViaChain,
								Message:            fmt.Sprintf("Unauthorized transitive access to %s", resourceID),
								Remediation:        "Remove impersonation permission in chain",
							})
						}
					}
				}
			}
		}
	}

	return violations
}

// validateEffectiveAccess validates effective access policies (direct + transitive)
func (v *PolicyValidator) validateEffectiveAccess(policy *Policy) []Violation {
	violations := []Violation{}
	effective := policy.EffectiveAccess

	if effective == nil {
		return violations
	}

	// Collect all principals with effective access (direct or transitive)
	effectiveAccess := make(map[string]map[string]bool) // resourceID -> principal -> true

	// Add direct access
	for principal, data := range v.directAccess {
		for resourceID, meta := range data.ResourceAccess {
			if !MatchesResourcePattern(resourceID, effective.Selector.ResourcePattern) {
				continue
			}
			if effective.Selector.ResourceType != "" && effective.Selector.ResourceType != "*" && meta.Type != effective.Selector.ResourceType {
				continue
			}

			if effectiveAccess[resourceID] == nil {
				effectiveAccess[resourceID] = make(map[string]bool)
			}
			effectiveAccess[resourceID][principal] = true
		}
	}

	// Add transitive access if validation enabled
	if effective.ValidateEffectiveAccess {
		for principal := range v.directAccess {
			email := ExtractPrincipalEmail(principal)
			transitiveAccess := analyzer.AnalyzeTransitiveAccess(email, v.directAccess, v.impGraph)

			if transitiveAccess == nil {
				continue
			}

			for resourceID := range transitiveAccess.TransitiveAccess {
				if !MatchesResourcePattern(resourceID, effective.Selector.ResourcePattern) {
					continue
				}

				if effectiveAccess[resourceID] == nil {
					effectiveAccess[resourceID] = make(map[string]bool)
				}
				effectiveAccess[resourceID][principal] = true
			}
		}
	}

	// Validate effective access
	for resourceID, principals := range effectiveAccess {
		for principal := range principals {
			// Check if principal is allowed
			isAllowed := len(effective.AllowedEffectivePrincipals) == 0 || IsPrincipalIn(principal, effective.AllowedEffectivePrincipals)
			isForbidden := IsPrincipalIn(principal, effective.ForbiddenEffectivePrincipals)

			if isForbidden || !isAllowed {
				// Determine if it's direct or transitive
				isDirect := false
				if data, exists := v.directAccess[principal]; exists {
					if _, hasResource := data.ResourceAccess[resourceID]; hasResource {
						isDirect = true
					}
				}

				accessType := "transitive"
				if isDirect {
					accessType = "direct"
				}

				violations = append(violations, Violation{
					PolicyName:    policy.Name,
					ViolationType: ViolationTypeEffectiveAccess,
					Severity:      policy.Severity,
					Principal:     principal,
					Resource:      resourceID,
					Message:       fmt.Sprintf("Unauthorized effective access (%s) to resource", accessType),
					Remediation:   "Remove principal access or update policy",
				})
			}
		}
	}

	return violations
}
