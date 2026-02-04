package policy

// PolicyConfig represents the complete policy configuration
type PolicyConfig struct {
	CloudProvider string   `yaml:"cloud_provider"`
	Policies      []Policy `yaml:"policies"`
}

// Policy represents a single policy definition
type Policy struct {
	Name        string     `yaml:"name"`
	Type        PolicyType `yaml:"type"`
	Description string     `yaml:"description"`
	Severity    Severity   `yaml:"severity"`

	// Type-specific fields
	RoleRestriction         *RoleRestrictionPolicy   `yaml:"role_restriction,omitempty"`
	Persona                 *PersonaPolicy           `yaml:"persona,omitempty"`
	ResourceAccess          *ResourceAccessPolicy    `yaml:"resource_access,omitempty"`
	SeparationOfDuty        *SeparationOfDutyPolicy  `yaml:"separation_of_duty,omitempty"`
	ImpersonationEscalation *ImpersonationEscalation `yaml:"impersonation_escalation,omitempty"`
	EffectiveAccess         *EffectiveAccessPolicy   `yaml:"effective_access,omitempty"`
}

// PolicyType defines the type of policy
type PolicyType string

const (
	PolicyTypeRoleRestriction         PolicyType = "role_restriction"
	PolicyTypePersona                 PolicyType = "persona"
	PolicyTypeResourceAccess          PolicyType = "resource_access"
	PolicyTypeSeparationOfDuty        PolicyType = "separation_of_duty"
	PolicyTypeImpersonationEscalation PolicyType = "impersonation_escalation"
	PolicyTypeEffectiveAccess         PolicyType = "effective_access"
)

// Severity levels
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// RoleRestrictionPolicy restricts which roles principals can have
type RoleRestrictionPolicy struct {
	Selector     PrincipalSelector `yaml:"selector"`
	AllowedRoles []string          `yaml:"allowed_roles"`
	DeniedRoles  []string          `yaml:"denied_roles"`
}

// PersonaPolicy defines required and forbidden bindings for a persona
type PersonaPolicy struct {
	PersonaName              string                 `yaml:"persona_name"`
	Principals               []string               `yaml:"principals"`
	RequiredBindings         []RequiredBinding      `yaml:"required_bindings"`
	ForbiddenBindings        []ForbiddenBinding     `yaml:"forbidden_bindings"`
	AllowAdditionalAccess    bool                   `yaml:"allow_additional_access"`
	ValidateTransitiveAccess bool                   `yaml:"validate_transitive_access"`
	TransitiveConstraints    *TransitiveConstraints `yaml:"transitive_constraints,omitempty"`
}

// ResourceAccessPolicy controls which principals can access resources
type ResourceAccessPolicy struct {
	Selector                 ResourceSelector    `yaml:"selector"`
	AllowedPrincipals        []string            `yaml:"allowed_principals"`
	AllowedRolesPerPrincipal map[string][]string `yaml:"allowed_roles_per_principal,omitempty"`
	ValidateEffectiveAccess  bool                `yaml:"validate_effective_access"`
}

// SeparationOfDutyPolicy prevents conflicting role combinations
type SeparationOfDutyPolicy struct {
	ConflictingRoles [][]string `yaml:"conflicting_roles"`
	Scope            string     `yaml:"scope"` // "per_principal" or "per_resource"
}

// ImpersonationEscalation prevents privilege escalation through impersonation
type ImpersonationEscalation struct {
	ForbiddenEscalations []EscalationRule `yaml:"forbidden_escalations"`
}

// EffectiveAccessPolicy validates complete access including transitive
type EffectiveAccessPolicy struct {
	Selector                     ResourceSelector `yaml:"selector"`
	ValidateEffectiveAccess      bool             `yaml:"validate_effective_access"`
	AllowedEffectivePrincipals   []string         `yaml:"allowed_effective_principals"`
	ForbiddenEffectivePrincipals []string         `yaml:"forbidden_effective_principals"`
}

// PrincipalSelector selects principals to apply policy to
type PrincipalSelector struct {
	PrincipalPattern string `yaml:"principal_pattern"`
}

// ResourceSelector selects resources to apply policy to
type ResourceSelector struct {
	ResourcePattern string `yaml:"resource_pattern"`
	ResourceType    string `yaml:"resource_type"`
}

// RequiredBinding defines a binding that must exist
type RequiredBinding struct {
	ResourcePattern string `yaml:"resource_pattern"`
	ResourceType    string `yaml:"resource_type"`
	Role            string `yaml:"role"`
}

// ForbiddenBinding defines a binding that must not exist
type ForbiddenBinding struct {
	ResourcePattern string `yaml:"resource_pattern"`
	ResourceType    string `yaml:"resource_type,omitempty"`
	Role            string `yaml:"role"`
}

// TransitiveConstraints defines constraints on transitive access
type TransitiveConstraints struct {
	MaxImpersonationDepth        int               `yaml:"max_impersonation_depth"`
	ForbiddenTransitiveRoles     []string          `yaml:"forbidden_transitive_roles"`
	ForbiddenTransitiveResources []ResourcePattern `yaml:"forbidden_transitive_resources"`
	AllowedImpersonationTargets  []string          `yaml:"allowed_impersonation_targets"`
}

// ResourcePattern defines a resource pattern for matching
type ResourcePattern struct {
	ResourcePattern string `yaml:"resource_pattern"`
	ResourceType    string `yaml:"resource_type,omitempty"`
}

// EscalationRule defines a forbidden privilege escalation
type EscalationRule struct {
	FromRolePattern      string `yaml:"from_role_pattern,omitempty"`
	ToRolePattern        string `yaml:"to_role_pattern,omitempty"`
	FromPrincipalPattern string `yaml:"from_principal_pattern,omitempty"`
	ToPrincipalPattern   string `yaml:"to_principal_pattern,omitempty"`
	ToResourcePattern    string `yaml:"to_resource_pattern,omitempty"`
	Via                  string `yaml:"via"` // "impersonation"
}

// Violation represents a policy violation
type Violation struct {
	PolicyName         string
	ViolationType      ViolationType
	Severity           Severity
	Principal          string
	Resource           string
	Role               string
	Message            string
	ImpersonationChain []string
	Location           string // File and line where violation occurs
	Remediation        string
}

// ViolationType categorizes violations
type ViolationType string

const (
	ViolationTypeForbiddenRole         ViolationType = "forbidden_role"
	ViolationTypeMissingRole           ViolationType = "missing_role"
	ViolationTypeUnauthorizedPrincipal ViolationType = "unauthorized_principal"
	ViolationTypeConflictingRoles      ViolationType = "conflicting_roles"
	ViolationTypeTransitiveRole        ViolationType = "transitive_role"
	ViolationTypeTransitiveResource    ViolationType = "transitive_resource"
	ViolationTypeImpersonationDepth    ViolationType = "impersonation_depth"
	ViolationTypePrivilegeEscalation   ViolationType = "privilege_escalation"
	ViolationTypeEffectiveAccess       ViolationType = "effective_access"
)

// ValidationReport summarizes policy validation results
type ValidationReport struct {
	TotalPolicies      int
	TotalViolations    int
	ErrorCount         int
	WarningCount       int
	InfoCount          int
	Violations         []Violation
	CompliantPolicies  []string
	PrincipalsAnalyzed int
	MaxChainDepth      int
	HighRiskFindings   []string
}
