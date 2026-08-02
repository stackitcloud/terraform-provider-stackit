package tfwriteid

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"
)

const (
	analyzerName = "tfwriteid"
)

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "Ensures that ID attributes are written to the state using TODO in every resource/datasource CRUD method before a wait handler from the STACKIT SDK is called",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	const (
		utilPkg      = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
		requiredFunc = "SetAndLogStateFields" // The util function which is used to store the ID fields to the state
	)

	inspectNode := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Filter only for function declarations (Terraform CRUD methods)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspectNode.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		if !lintutils.IsTerraformLifecycleMethodCreateOnlyTODO(funcDecl) {
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
			if pkgPath == utilPkg && calledFuncName == requiredFunc {
				hasCalledUtil = true
			}

			// Check if we've hit a STACKIT SDK wait handler call before the util function
			callsWait := isWaitCall(pass.TypesInfo, call, calledFuncName)
			if callsWait && !hasCalledUtil {
				pass.Reportf(
					call.Pos(),
					"%s: call to wait handler from %s must happen AFTER %s.%s is called in %s %s",
					analyzerName, lintutils.StackitSdkModulePrefix, utilPkg, requiredFunc, funcDecl.Name.Name, pkgPath,
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

func isWaitCall(info *types.Info, call *ast.CallExpr, calledFuncName string) bool {
	// must be a selector like handler.WaitWithContext()
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	// serviceenablement itself is used in other resources and does not have an ID itself
	if waiterCall, ok := sel.X.(*ast.CallExpr); ok {
		_, waiterFuncName := lintutils.GetCallInfo(waiterCall, info)
		if waiterFuncName == "EnableServiceWaitHandler" {
			return false
		}
	}
	obj := info.Uses[sel.Sel]
	if obj == nil {
		return false
	}
	sig, ok := obj.Type().(*types.Signature)
	if !ok {
		return false
	}
	// get full receiver type
	recv := sig.Recv()
	if recv == nil {
		return false
	}
	recvType := recv.Type()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok {
		return false
	}
	if named.Obj().Pkg() == nil {
		return false
	}
	recvPkgName := named.Obj().Pkg().Path()
	recvTypeName := named.Obj().Name()
	// must be WaitWithContext on a wait.AsyncActionHandler receiver
	return recvPkgName == "github.com/stackitcloud/stackit-sdk-go/core/wait" &&
		recvTypeName == "AsyncActionHandler" &&
		calledFuncName == "WaitWithContext"
}
