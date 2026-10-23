package lintutils

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// helper function to parse a string into an *ast.FuncDecl
func getFuncDecl(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()

	// Parse the source code snippet
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("failed to parse src: %v", err)
	}

	// Find and return the first function declaration
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fn
		}
	}

	t.Fatal("no function declaration found in source snippet")
	return nil
}

func TestIsTerraformLifecycleMethod(t *testing.T) {
	tests := []struct {
		name              string
		src               string // The Go code snippet containing the function to test
		methodNamesFilter []string
		expected          bool
	}{
		{
			name:     "Valid Create method (default filter)",
			src:      `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {}`,
			expected: true,
		},
		{
			name:              "Valid Create method (custom filter)",
			src:               `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {}`,
			methodNamesFilter: []string{"Create"},
			expected:          true,
		},
		{
			name:     "Valid Read method (default filter)",
			src:      `package main; func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {}`,
			expected: true,
		},
		{
			name:              "Valid Configure method (custom filter)",
			src:               `package main; func (r *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {}`,
			methodNamesFilter: []string{"Configure"},
			expected:          true,
		},
		{
			name:     "Invalid: No receiver (regular function)",
			src:      `package main; func Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {}`,
			expected: false,
		},
		{
			name:     "Invalid: Method name not in default filter",
			src:      `package main; func (r *Resource) Validate(ctx context.Context, req resource.ValidateRequest, resp *resource.ValidateResponse) {}`,
			expected: false,
		},
		{
			name:              "Invalid: Method name not in custom filter",
			src:               `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {}`,
			methodNamesFilter: []string{"Update", "Delete"}, // "Create" is missing
			expected:          false,
		},
		{
			name:     "Invalid: Wrong parameter count (only 2 parameters)",
			src:      `package main; func (r *Resource) Create(req resource.CreateRequest, resp *resource.CreateResponse) {}`,
			expected: false,
		},
		{
			name:     "Invalid: Wrong parameter count (4 parameters)",
			src:      `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, extra string) {}`,
			expected: false,
		},
		{
			name:     "Invalid: Parameter 2 does not end in Request",
			src:      `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateModel, resp *resource.CreateResponse) {}`,
			expected: false,
		},
		{
			name:     "Invalid: Parameter 3 does not end in Response",
			src:      `package main; func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateState) {}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			funcDecl := getFuncDecl(t, tt.src)

			actual := IsTerraformLifecycleMethod(funcDecl, tt.methodNamesFilter...)
			if actual != tt.expected {
				t.Errorf("IsTerraformLifecycleMethod() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestHasSuffixType(t *testing.T) {
	tests := []struct {
		name     string
		field    *ast.Field
		suffix   string
		expected bool
	}{
		{
			name: "SelectorExpr matching suffix",
			// Represents: resource.CreateRequest
			field: &ast.Field{
				Type: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "resource"},
					Sel: &ast.Ident{Name: "CreateRequest"},
				},
			},
			suffix:   "Request",
			expected: true,
		},
		{
			name: "SelectorExpr not matching suffix",
			// Represents: resource.CreateResponse
			field: &ast.Field{
				Type: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "resource"},
					Sel: &ast.Ident{Name: "CreateResponse"},
				},
			},
			suffix:   "Request",
			expected: false,
		},
		{
			name: "StarExpr wrapping SelectorExpr matching suffix",
			// Represents: *resource.CreateResponse
			field: &ast.Field{
				Type: &ast.StarExpr{
					X: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "resource"},
						Sel: &ast.Ident{Name: "CreateResponse"},
					},
				},
			},
			suffix:   "Response",
			expected: true,
		},
		{
			name: "StarExpr wrapping SelectorExpr not matching suffix",
			// Represents: *resource.CreateResponse
			field: &ast.Field{
				Type: &ast.StarExpr{
					X: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "resource"},
						Sel: &ast.Ident{Name: "CreateResponse"},
					},
				},
			},
			suffix:   "Request",
			expected: false,
		},
		{
			name: "Unsupported type (Ident) returns false",
			// Represents a built-in type: string
			field: &ast.Field{
				Type: &ast.Ident{Name: "string"},
			},
			suffix:   "Request",
			expected: false,
		},
		{
			name: "StarExpr wrapping non-SelectorExpr (Ident) returns false",
			// Represents a pointer to a built-in type: *string
			field: &ast.Field{
				Type: &ast.StarExpr{
					X: &ast.Ident{Name: "string"},
				},
			},
			suffix:   "Request",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSuffixType(tt.field, tt.suffix)
			if result != tt.expected {
				t.Errorf("hasSuffixType() = %v, want %v", result, tt.expected)
			}
		})
	}
}
