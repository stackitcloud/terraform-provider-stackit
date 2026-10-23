package tfacl

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "tfacl",
	Doc:  "Prevents the usage of 'acls' in terraform plugin framework field names; 'acl' should be used instead.",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {

			// 1. Check Go Struct Field Names and Struct Tags
			case *ast.Field:
				// Check the Go field name (e.g., `Acls types.List`)
				for _, name := range node.Names {
					if strings.ToLower(name.Name) == "acls" {
						pass.Reportf(
							name.Pos(),
							"field name %q uses 'acls'; use 'acl' instead",
							name.Name,
						)
					}
				}

				// Check struct tags (e.g., `tfsdk:"acls"`)
				if node.Tag != nil {
					// We look for exactly "acls" inside the tag string
					if strings.Contains(node.Tag.Value, `"acls"`) {
						pass.Reportf(
							node.Tag.Pos(),
							"struct tag %s contains 'acls'; use 'acl' instead",
							node.Tag.Value,
						)
					}
				}

			// 2. Check Schema Map Keys
			case *ast.KeyValueExpr:
				// Check if the key in a map or composite literal is the exact string "acls"
				// (e.g., Attributes: map[string]schema.Attribute{"acls": ...})
				if key, ok := node.Key.(*ast.BasicLit); ok && key.Kind == token.STRING {
					if key.Value == `"acls"` {
						pass.Reportf(
							key.Pos(),
							"schema attribute key %s uses 'acls'; use 'acl' instead",
							key.Value,
						)
					}
				}
			}
			return true // Continue traversing the AST
		})
	}

	return nil, nil
}

func init() {
	register.Plugin("tfacl", New)
}

func New(settings any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	// LoadModeSyntax is required because we need to inspect the AST (Syntax trees)
	return register.LoadModeSyntax
}
