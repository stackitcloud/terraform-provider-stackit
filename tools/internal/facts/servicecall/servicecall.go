// Package servicecall identifies functions that make STACKIT SDK service calls.
package servicecall

import (
	"go/ast"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
	"golang.org/x/tools/refactor/satisfy"

	"github.com/stackitcloud/terraform-provider-stackit/tools/internal/lintutils"
)

var Analyzer = &analysis.Analyzer{
	Name:     "servicecall",
	Doc:      "Publishes a fact for functions that call a STACKIT SDK service",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{
		new(IsServiceCall),
	},
	ResultType: reflect.TypeOf((*Result)(nil)),
	Run:        run,
}

// IsServiceCall is a fact indicating that a function calls a STACKIT SDK service.
type IsServiceCall struct{}

func (*IsServiceCall) AFact() {}

func (*IsServiceCall) String() string { return "serviceCall" }

// Result provides the service-call functions identified while analyzing a package.
type Result struct {
	functions map[*types.Func]struct{}
}

// HasServiceCall reports whether fn is marked as making a STACKIT SDK service call.
func (r *Result) HasServiceCall(fn *types.Func) bool {
	_, ok := r.functions[fn.Origin()]
	return ok
}

// HasServiceCall reports whether call invokes a marked service-call function
// directly or receives one as a function reference argument.
func HasServiceCall(call *ast.CallExpr, info *types.Info, result *Result) bool {
	callee, _ := typeutil.Callee(info, call).(*types.Func)
	if callee != nil && result.HasServiceCall(callee) {
		return true
	}

	for _, argument := range call.Args {
		if referencedFunction := functionReference(argument, info); referencedFunction != nil && result.HasServiceCall(referencedFunction) {
			return true
		}
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// callers maps a callee to the functions that call it. Once a callee is
	// known to call a service, the fact is propagated to all of its callers.
	callers := make(map[*types.Func]map[*types.Func]struct{})
	serviceCalls := make(map[*types.Func]struct{})
	var serviceCallers []*types.Func

	// A function that receives a known service-call function will invoke that
	// function indirectly, so it is itself a service-call function.
	markReferencedServiceCall := func(caller, referencedFunction *types.Func) {
		if referencedFunction == nil || !isKnownServiceCall(pass, nil, referencedFunction) {
			return
		}
		serviceCalls[referencedFunction.Origin()] = struct{}{}
		serviceCallers = append(serviceCallers, caller)
	}

	inspect.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		decl := n.(*ast.FuncDecl)
		if decl.Body == nil {
			return
		}

		caller, _ := pass.TypesInfo.Defs[decl.Name].(*types.Func)
		if caller == nil {
			return
		}
		caller = caller.Origin()

		ast.Inspect(decl.Body, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				// A call in a closure does not belong to the enclosing function.
				return false
			}

			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, argument := range call.Args {
				markReferencedServiceCall(caller, functionReference(argument, pass.TypesInfo))
			}

			callee, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
			if callee == nil {
				return true
			}
			callee = callee.Origin()

			if isKnownServiceCall(pass, call, callee) {
				serviceCalls[callee] = struct{}{}
				serviceCallers = append(serviceCallers, caller)
				return true
			}
			if callee.Pkg() == pass.Pkg {
				if callers[callee] == nil {
					callers[callee] = make(map[*types.Func]struct{})
				}
				callers[callee][caller] = struct{}{}
			}
			return true
		})
	})

	addInterfaceEdges(pass, callers)

	marked := make(map[*types.Func]struct{})
	var propagate func(*types.Func)
	propagate = func(fn *types.Func) {
		if _, ok := marked[fn]; ok {
			return
		}
		marked[fn] = struct{}{}
		pass.ExportObjectFact(fn, new(IsServiceCall))
		for caller := range callers[fn] {
			propagate(caller)
		}
	}
	for _, fn := range serviceCallers {
		propagate(fn)
	}
	for fn := range marked {
		serviceCalls[fn] = struct{}{}
	}

	return &Result{functions: serviceCalls}, nil
}

// addInterfaceEdges models calls through interface methods as calls to each
// implementation established by an assignment in the current package. This
// permits facts from concrete SDK service methods to propagate through the
// request's interface field to its Execute method.
func addInterfaceEdges(pass *analysis.Pass, callers map[*types.Func]map[*types.Func]struct{}) {
	var finder satisfy.Finder
	finder.Find(pass.TypesInfo, pass.Files)

	for assignment := range finder.Result {
		iface := assignment.LHS.Underlying().(*types.Interface)
		for method := range iface.Methods() {
			// Facts can only be exported for objects in the current package.
			if method.Pkg() != pass.Pkg {
				continue
			}

			implementation, _, _ := types.LookupFieldOrMethod(assignment.RHS, false, pass.Pkg, method.Name())
			implementationFunc, ok := implementation.(*types.Func)
			if !ok {
				continue
			}

			implementationFunc = implementationFunc.Origin()
			method = method.Origin()
			if callers[implementationFunc] == nil {
				callers[implementationFunc] = make(map[*types.Func]struct{})
			}
			callers[implementationFunc][method] = struct{}{}
		}
	}
}

// functionReference returns the function denoted by expr, if expr is a
// statically identifiable function value.
func functionReference(expr ast.Expr, info *types.Info) *types.Func {
	switch expr := expr.(type) {
	case *ast.Ident:
		fn, _ := info.ObjectOf(expr).(*types.Func)
		return fn
	case *ast.SelectorExpr:
		fn, _ := info.ObjectOf(expr.Sel).(*types.Func)
		return fn
	case *ast.ParenExpr:
		return functionReference(expr.X, info)
	case *ast.IndexExpr:
		return functionReference(expr.X, info)
	case *ast.IndexListExpr:
		return functionReference(expr.X, info)
	}
	return nil
}

func isKnownServiceCall(pass *analysis.Pass, call *ast.CallExpr, fn *types.Func) bool {
	return isSDKCallAPI(fn) ||
		(call != nil && lintutils.IsWaitCall(pass.TypesInfo, call, fn.Name())) ||
		hasServiceCallFact(pass, fn)
}

func hasServiceCallFact(pass *analysis.Pass, fn *types.Func) bool {
	if fn.Pkg() == pass.Pkg {
		return false
	}
	var fact IsServiceCall
	return pass.ImportObjectFact(fn, &fact)
}

func isSDKCallAPI(fn *types.Func) bool {
	if fn.Name() != "callAPI" || fn.Pkg() == nil || !strings.HasPrefix(fn.Pkg().Path(), lintutils.StackitSdkModulePrefix) {
		return false
	}

	receiver := fn.Signature().Recv()
	if receiver == nil {
		return false
	}
	pointer, ok := receiver.Type().(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := pointer.Elem().(*types.Named)
	return ok && named.Obj().Name() == "APIClient" && named.Obj().Pkg() == fn.Pkg()
}
