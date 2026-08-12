package tfmodifyplan

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "tfmodifyplan",
	Doc:  "Ensures every resource with a region field implements the ModifyPlan method.",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	// Iterate over all parsed Go files in the package
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			// 1. Find all method declarations named "Schema"
			fn, ok := node.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Schema" || fn.Recv == nil || len(fn.Recv.List) == 0 {
				return true
			}

			// 2. Search the AST body of the Schema method for a "region" attribute key
			hasRegionAttr := false
			ast.Inspect(fn.Body, func(innerNode ast.Node) bool {
				kv, ok := innerNode.(*ast.KeyValueExpr)
				if !ok {
					return true
				}

				keyLit, ok := kv.Key.(*ast.BasicLit)
				if ok && keyLit.Kind == token.STRING {
					// String literals in the AST include their quotes, so we check both styles
					if keyLit.Value == `"region"` || keyLit.Value == "`region`" {
						hasRegionAttr = true
						return false // Stop traversing this subtree, we found what we need
					}
				}
				return true
			})

			if !hasRegionAttr {
				return true
			}

			// 3. Extract the receiver's underlying type name (e.g., `*MyResource` -> `MyResource`)
			recvExpr := fn.Recv.List[0].Type
			if star, ok := recvExpr.(*ast.StarExpr); ok {
				recvExpr = star.X
			}

			ident, ok := recvExpr.(*ast.Ident)
			if !ok {
				return true
			}

			// 4. Resolve the AST identifier to its actual type representation via TypesInfo
			obj := pass.TypesInfo.ObjectOf(ident)
			if obj == nil {
				return true
			}

			named, ok := obj.Type().(*types.Named)
			if !ok {
				return true
			}

			ptrType := types.NewPointer(named)

			// 5. Exclude Data Sources by ensuring the type implements 'Create'.
			// (Resources have Create, Update, Delete. Data Sources only have Read).
			if !hasMethod(named, "Create") && !hasMethod(ptrType, "Create") {
				return true
			}

			// 6. Check if the ModifyPlan method is present on the value or pointer receiver
			if !hasMethod(named, "ModifyPlan") && !hasMethod(ptrType, "ModifyPlan") {
				pass.Reportf(ident.Pos(), "'%s' defines a 'region' attribute in its Schema but does not implement the ModifyPlan method", ident.Name)
			}

			return true
		})
	}

	return nil, nil
}

// hasMethod checks if a given type's method set contains a specific method name.
func hasMethod(t types.Type, methodName string) bool {
	mset := types.NewMethodSet(t)
	for method := range mset.Methods() {
		if method.Obj().Name() == methodName {
			return true
		}
	}
	return false
}

func init() {
	register.Plugin("tfmodifyplan", New)
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
