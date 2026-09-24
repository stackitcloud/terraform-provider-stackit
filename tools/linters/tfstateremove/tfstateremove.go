package tfstateremove

import (
	"go/ast"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"
)

const analyzerName = "tfstateremove"

var Analyzer = &analysis.Analyzer{
	Name:     analyzerName,
	Doc:      "Ensures resource Read methods can remove resources from state",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// run checks each Terraform Framework resource Read method in these steps:
//  1. Identify methods with the Terraform Framework Read request and response types
//  2. Scan the method body for a call to the response parameter's State.RemoveResource
//  3. Report the method if no such call appears anywhere in its body
func run(pass *analysis.Pass) (any, error) {
	inspectNode := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectNode.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if !isResourceReadMethod(fn, pass.TypesInfo) || len(fn.Type.Params.List[2].Names) == 0 {
			return
		}

		response := fn.Type.Params.List[2].Names[0].Name
		if !hasRemoveResource(fn.Body, response) {
			pass.Reportf(fn.Name.Pos(), "%s: resource Read must call %s.State.RemoveResource(ctx)", analyzerName, response)
		}
	})

	return nil, nil
}

func isResourceReadMethod(fn *ast.FuncDecl, info *types.Info) bool {
	if !lintutils.IsTerraformLifecycleMethod(fn, "Read") {
		return false
	}

	req := namedType(info.TypeOf(fn.Type.Params.List[1].Type))
	resp := namedType(info.TypeOf(fn.Type.Params.List[2].Type))
	return req != nil && resp != nil &&
		req.Obj().Name() == "ReadRequest" && resp.Obj().Name() == "ReadResponse" &&
		req.Obj().Pkg() != nil && resp.Obj().Pkg() != nil &&
		req.Obj().Pkg().Path() == "github.com/hashicorp/terraform-plugin-framework/resource" &&
		resp.Obj().Pkg().Path() == "github.com/hashicorp/terraform-plugin-framework/resource"
}

func namedType(t types.Type) *types.Named {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, _ := t.(*types.Named)
	return named
}

func hasRemoveResource(block *ast.BlockStmt, response string) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		method, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || method.Sel.Name != "RemoveResource" {
			return true
		}
		state, ok := method.X.(*ast.SelectorExpr)
		if !ok || state.Sel.Name != "State" {
			return true
		}
		root, ok := state.X.(*ast.Ident)
		found = ok && root.Name == response
		return !found
	})
	return found
}

func init() {
	register.Plugin(analyzerName, New)
}

func New(_ any) (register.LinterPlugin, error) { return &plugin{}, nil }

type plugin struct{}

func (*plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (*plugin) GetLoadMode() string { return register.LoadModeTypesInfo }
