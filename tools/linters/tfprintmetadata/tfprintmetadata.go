package tfprintmetadata

import (
	"go/ast"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"
)

const analyzerName = "tfprintmetadata"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "Ensures that testutil.PrintMetadata is always called as the first thing in all test cases starting with the name TestAcc",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	targetFuncPrefix = "TestAcc"
	targetFuncName   = "PrintMetadata"
	targetPkg        = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func run(pass *analysis.Pass) (any, error) {
	inspectNode, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok || inspectNode == nil {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspectNode.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		if !isTestAccCase(funcDecl) {
			return
		}

		if !hasPrintMetadataFirst(funcDecl, pass) {
			pass.Reportf(
				funcDecl.Name.Pos(),
				"%s: test case %s must call testutil.PrintMetadata as the first statement",
				analyzerName,
				funcDecl.Name.Name,
			)
		}
	})

	return nil, nil
}

func isTestAccCase(fn *ast.FuncDecl) bool {
	// 1. Must be a standalone function, not a method (no receiver)
	if fn.Recv != nil {
		return false
	}

	// 2. Must start with "TestAcc"
	if !strings.HasPrefix(fn.Name.Name, targetFuncPrefix) {
		return false
	}

	return true
}

func hasPrintMetadataFirst(fn *ast.FuncDecl, pass *analysis.Pass) bool {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return false
	}

	firstStmt := fn.Body.List[0]
	return isPrintMetadataCall(firstStmt, pass)
}

func isPrintMetadataCall(stmt ast.Stmt, pass *analysis.Pass) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	// 1. If type information is available, resolve the package and function name.
	if pass != nil && pass.TypesInfo != nil {
		pkgPath, calledFuncName := lintutils.GetCallInfo(call, pass.TypesInfo)
		if pkgPath != "" {
			return calledFuncName == targetFuncName &&
				(pkgPath == targetPkg || strings.HasSuffix(pkgPath, "/testutil") || pkgPath == "testutil")
		}
	}

	// 2. Fallback to syntax inspection (for LoadModeSyntax, incomplete types, or tests)
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == targetFuncName {
			if ident, ok := sel.X.(*ast.Ident); ok {
				return ident.Name == "testutil"
			}
		}
	}

	return false
}

func init() {
	register.Plugin(analyzerName, New)
}

func New(_ any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (*plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (*plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
