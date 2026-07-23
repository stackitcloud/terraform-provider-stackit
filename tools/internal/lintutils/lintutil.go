package lintutils

import (
	"go/ast"
	"go/types"
	"strings"
)

const StackitSdkModulePrefix = "github.com/stackitcloud/stackit-sdk-go"

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
// lifecycle implementation (CRUD)
func IsTerraformLifecycleMethod(funcDecl *ast.FuncDecl) bool {
	// 1. Must be a method (have a receiver)
	if funcDecl.Recv == nil {
		return false
	}

	// 2. Check if it is a Terraform Provider Framework CRUD method
	name := funcDecl.Name.Name
	if name != "Create" && name != "Read" && name != "Update" && name != "Delete" {
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

// IsTerraformLifecycleMethod is used to check whether an AST function declaration is a Terraform resource
// lifecycle implementation (CRUD)
func IsTerraformLifecycleMethodCreateOnlyTODO(funcDecl *ast.FuncDecl) bool {
	// 1. Must be a method (have a receiver)
	if funcDecl.Recv == nil {
		return false
	}

	// 2. Check if it is a Terraform Provider Framework CRUD method
	name := funcDecl.Name.Name
	if name != "Create" {
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
