package parser

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// ConfigTraverser provides methods to resolve expressions by looking up resources in the configuration.
type ConfigTraverser struct {
	Files         []*hcl.File
	Variables     map[string]cty.Value
	ScopedContext *hcl.EvalContext // For loop variables like count.index, each.key, each.value
}

// NewConfigTraverser creates a new traverser.
func NewConfigTraverser(files []*hcl.File, vars map[string]cty.Value) *ConfigTraverser {
	return &ConfigTraverser{
		Files:     files,
		Variables: vars,
	}
}

// ResolveExpression attempts to resolve an HCL expression to a cty.Value.
// It handles:
// 1. Literal values
// 2. Variables (var.x)
// 3. Resource references (type.name.attr)
func (t *ConfigTraverser) ResolveExpression(expr hcl.Expression) (cty.Value, error) {
	// 1. Try resolving with variables context first (catches literals and vars)
	ctx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(t.Variables),
		},
		Functions: map[string]function.Function{
			"length": stdlib.LengthFunc,
		},
	}

	// Merge in scoped context if present (for count.index, each.key, each.value)
	if t.ScopedContext != nil {
		for k, v := range t.ScopedContext.Variables {
			ctx.Variables[k] = v
		}
		// Also merge functions if any
		if t.ScopedContext.Functions != nil {
			for k, v := range t.ScopedContext.Functions {
				ctx.Functions[k] = v
			}
		}
	}

	val, diags := expr.Value(ctx)
	if !diags.HasErrors() {
		return val, nil
	}

	// 2. If valid failure, check if it's a traversal (reference)
	if syntaxExpr, ok := expr.(hclsyntax.Expression); ok {
		_ = syntaxExpr
	}

	traversal, diags := hcl.AbsTraversalForExpr(expr)
	if diags.HasErrors() {
		return cty.NilVal, fmt.Errorf("expression is not a simple traversal and could not be evaluated with variables")
	}

	// 3. Handle Resource Reference: Type.Name.Attr
	if len(traversal) < 3 {
		return cty.NilVal, fmt.Errorf("traversal too short to be a resource reference")
	}

	// Root = Type, Next = Name, Next = Attr
	// Note: This is a simplification. References can be intricate.

	parts := make([]string, 0, len(traversal))
	for _, step := range traversal {
		switch s := step.(type) {
		case hcl.TraverseRoot:
			parts = append(parts, s.Name)
		case hcl.TraverseAttr:
			parts = append(parts, s.Name)
		case hcl.TraverseIndex:
			// We don't support index lookup yet?
			// Maybe later.
		}
	}

	if len(parts) < 3 {
		return cty.NilVal, fmt.Errorf("unsupported traversal format")
	}

	resType := parts[0]
	resName := parts[1]
	attrName := parts[2]

	// Look up the resource
	return t.LookupResourceAttribute(resType, resName, attrName)
}

// LookupResourceAttribute scans files for a resource block and extracts specifically the attribute.
// It recursively calls ResolveExpression on the found attribute.
func (t *ConfigTraverser) LookupResourceAttribute(resType, resName, attrName string) (cty.Value, error) {
	for _, file := range t.Files {
		// We need to parse body to blocks again... or better, assume files are already parsed and we inspect the body?
		// Since we have *hcl.File, we can inspect Body.

		// This is inefficient scanning but fine for MVP
		content, _, _ := file.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "resource", LabelNames: []string{"type", "name"}},
			},
		})

		for _, block := range content.Blocks {
			if block.Type == "resource" && len(block.Labels) == 2 {
				if block.Labels[0] == resType && block.Labels[1] == resName {
					// Found Resource!
					// Extract Attribute
					// We need to look for attrName.
					// Attributes are in block.Body.
					blockContent, _, _ := block.Body.PartialContent(&hcl.BodySchema{
						Attributes: []hcl.AttributeSchema{
							{Name: attrName, Required: false},
						},
					})

					if attr, exists := blockContent.Attributes[attrName]; exists {
						// Recursively resolve!
						return t.ResolveExpression(attr.Expr)
					} else {
						return cty.NilVal, fmt.Errorf("attribute %s not found in resource %s.%s", attrName, resType, resName)
					}
				}
			}
		}
	}

	return cty.NilVal, fmt.Errorf("resource %s.%s not found", resType, resName)
}

// WithScope returns a new ConfigTraverser with the given scoped context.
// This is used when processing loop iterations to provide count.index, each.key, each.value.
func (t *ConfigTraverser) WithScope(scopedCtx *hcl.EvalContext) *ConfigTraverser {
	return &ConfigTraverser{
		Files:         t.Files,
		Variables:     t.Variables,
		ScopedContext: scopedCtx,
	}
}
