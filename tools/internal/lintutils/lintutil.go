package lintutils

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
)

const StackitSdkModulePrefix = "github.com/stackitcloud/stackit-sdk-go"

// IsWaitCall reports whether call executes an SDK async-action wait handler.
func IsWaitCall(info *types.Info, call *ast.CallExpr, calledFuncName string) bool {
	// Must be a selector like handler.WaitWithContext().
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	// serviceenablement itself is used in other resources and does not have an ID itself.
	if waiterCall, ok := sel.X.(*ast.CallExpr); ok {
		_, waiterFuncName := GetCallInfo(waiterCall, info)
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
	recv := sig.Recv()
	if recv == nil {
		return false
	}
	recvType := recv.Type()
	if ptr, ok := recvType.(*types.Pointer); ok {
		recvType = ptr.Elem()
	}
	named, ok := recvType.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	// Must be WaitWithContext on a wait.AsyncActionHandler receiver.
	return named.Obj().Pkg().Path() == "github.com/stackitcloud/stackit-sdk-go/core/wait" &&
		named.Obj().Name() == "AsyncActionHandler" &&
		calledFuncName == "WaitWithContext"
}

// GetCallInfo resolves the package path and function name of a CallExpr.
func GetCallInfo(call *ast.CallExpr, info *types.Info) (string, string) {
	var ident *ast.Ident

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		ident = fun
	case *ast.SelectorExpr:
		ident = fun.Sel
	default:
		return "", ""
	}

	obj := info.Uses[ident]
	if obj == nil {
		return "", ""
	}

	fn, ok := obj.(*types.Func)
	if !ok {
		return "", ""
	}

	pkg := fn.Pkg()
	if pkg == nil {
		return "", fn.Name() // Built-in functions
	}

	return pkg.Path(), fn.Name()
}

// IsTerraformLifecycleMethod is used to check whether an AST function declaration is a Terraform resource
// lifecycle implementation. If no methodNamesFilter values are given it checks for all methods (CRUD).
func IsTerraformLifecycleMethod(funcDecl *ast.FuncDecl, methodNamesFilter ...string) bool {
	// 1. Must be a method (have a receiver)
	if funcDecl.Recv == nil {
		return false
	}

	// 2. Check if it is a Terraform Provider Framework CRUD method
	if len(methodNamesFilter) == 0 {
		// use the default fallback values if no values were given
		methodNamesFilter = []string{"Create", "Read", "Update", "Delete"}
	}

	if !slices.Contains(methodNamesFilter, funcDecl.Name.Name) {
		return false
	}

	// 3. Ensure the method has exactly 3 parameters (e.g., ctx, req, resp)
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) != 3 {
		return false
	}

	// 4. Verify it's actually a Terraform framework method by checking if
	// param 2 ends with "Request" and param 3 ends with "Response"
	if !hasSuffixType(funcDecl.Type.Params.List[1], "Request") || !hasSuffixType(funcDecl.Type.Params.List[2], "Response") {
		return false
	}

	return true
}

// Helper to roughly check if a parameter type ends with a specific string (like "Request" or "Response")
func hasSuffixType(field *ast.Field, suffix string) bool {
	var typeName string
	switch t := field.Type.(type) {
	case *ast.SelectorExpr:
		typeName = t.Sel.Name // e.g., resource.CreateRequest
	case *ast.StarExpr:
		if sel, ok := t.X.(*ast.SelectorExpr); ok {
			typeName = sel.Sel.Name // e.g., *resource.CreateResponse
		}
	}
	return strings.HasSuffix(typeName, suffix)
}
