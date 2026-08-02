package tflogresponse

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/facts/servicecall"
	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"
)

const (
	analyzerName = "tflogresponse"
)

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "Ensures that core.LogResponse is called in every resource/datasource CRUD method after ctx.InitProviderContext was called and at least one STACKIT SDK call was made.",
	Requires: []*analysis.Analyzer{inspect.Analyzer, servicecall.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	const (
		utilPkg                 = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
		funcInitProviderContext = "InitProviderContext" // The util function that starts the sequence
		funcLogResponse         = "LogResponse"         // The util function that must follow
	)

	inspectNode := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	serviceCalls := pass.ResultOf[servicecall.Analyzer].(*servicecall.Result)

	// Filter only for function declarations (Terraform CRUD methods)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspectNode.Preorder(nodeFilter, func(n ast.Node) {
		funcDecl := n.(*ast.FuncDecl)

		if !lintutils.IsTerraformLifecycleMethod(funcDecl) {
			return
		}

		// State machine variables
		type state int
		const (
			stateLookingForInitProviderContextCall state = iota
			stateLookingForSdkOrLogResponseCall
		)

		currentState := stateLookingForInitProviderContextCall
		var posInitProviderContextCall ast.Node
		var foundIntermediateSdkModuleCall bool

		// Traverse the function body
		ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			pkgPath, calledFuncName := lintutils.GetCallInfo(call, pass.TypesInfo)
			if pkgPath == "" {
				return true
			}

			switch currentState {
			case stateLookingForInitProviderContextCall:
				if pkgPath == utilPkg && calledFuncName == funcInitProviderContext {
					currentState = stateLookingForSdkOrLogResponseCall
					posInitProviderContextCall = call
					foundIntermediateSdkModuleCall = false
				}

			case stateLookingForSdkOrLogResponseCall:
				if hasServiceCall(serviceCalls, pass, call) {
					foundIntermediateSdkModuleCall = true
				} else if pkgPath == utilPkg && calledFuncName == funcLogResponse {
					if !foundIntermediateSdkModuleCall {
						pass.Reportf(call.Pos(), "%s: invalid sequence: %s called without an intermediate call to %s after %s", analyzerName, funcLogResponse, lintutils.StackitSdkModulePrefix, funcInitProviderContext)
					}

					// reset the state for future findings
					currentState = stateLookingForInitProviderContextCall
				}
			}
			return true
		})

		// If we reach the end of the function and are still waiting for the LogResponse util func to be called
		if currentState == stateLookingForSdkOrLogResponseCall {
			pass.Reportf(posInitProviderContextCall.Pos(), "%s: invalid sequence: %s was called, but %s was never called afterwards", analyzerName, funcInitProviderContext, funcLogResponse)
		}
	})

	return nil, nil
}

func hasServiceCall(serviceCalls *servicecall.Result, pass *analysis.Pass, call *ast.CallExpr) bool {
	callee, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	return callee != nil && serviceCalls.HasServiceCall(callee)
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
