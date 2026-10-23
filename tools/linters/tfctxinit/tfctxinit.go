package tfctxinit

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/ast/inspector"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

const (
	analyzerName = "tfctxinit"
)

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "Ensures core.InitProviderContext is called before any SDK call in a Terraform resource lifecycle (CRUD) implementation",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	const (
		// This specific function must be called first
		requiredPkg  = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
		requiredFunc = "InitProviderContext"
	)

	inspectNode := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Filter only for function declarations (Terraform CRUD methods)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspectNode.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		if !lintutils.IsTerraformLifecycleMethod(funcDecl) {
			return
		}

		hasCalledUtil := false

		// Walk the function body AST sequentially
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true // Continue searching children
			}

			pkgPath, calledFuncName := lintutils.GetCallInfo(call, pass.TypesInfo)
			if pkgPath == "" {
				return true
			}

			// Check if we hit the required utility function
			if pkgPath == requiredPkg && calledFuncName == requiredFunc {
				hasCalledUtil = true
			}

			// Check if we've hit a STACKIT SDK call before the util function
			if strings.HasPrefix(pkgPath, lintutils.StackitSdkModulePrefix) && !hasCalledUtil {
				pass.Reportf(
					call.Pos(),
					"%s: call to %s must happen AFTER %s.%s is called in %s",
					analyzerName, lintutils.StackitSdkModulePrefix, requiredPkg, requiredFunc, funcDecl.Name.Name,
				)
			}

			return true
		})
	})

	return nil, nil
}

func init() {
	register.Plugin(analyzerName, New)
}

func New(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
