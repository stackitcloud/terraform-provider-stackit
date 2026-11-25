package tfprintmetadata

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "tfprintmetadata")
}

func parseFuncDecl(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse src: %v", err)
	}
	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fn
		}
	}
	t.Fatal("no function declaration found")
	return nil
}

func TestIsTestAccCase(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{
			name:     "valid TestAcc function",
			src:      `package test; func TestAccExample(t *testing.T) {}`,
			expected: true,
		},
		{
			name:     "method named TestAcc",
			src:      `package test; func (s *Suite) TestAccExample(t *testing.T) {}`,
			expected: false,
		},
		{
			name:     "regular unit test",
			src:      `package test; func TestExample(t *testing.T) {}`,
			expected: false,
		},
		{
			name:     "helper function",
			src:      `package test; func helper() {}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFuncDecl(t, tt.src)
			if actual := isTestAccCase(fn); actual != tt.expected {
				t.Errorf("isTestAccCase() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}

func TestHasPrintMetadataFirst(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		expected bool
	}{
		{
			name: "PrintMetadata is called first",
			src: `package test
func TestAccExample(t *testing.T) {
	testutil.PrintMetadata(t, testutil.AccTestMetadata{})
	t.Log("ok")
}`,
			expected: true,
		},
		{
			name: "PrintMetadata is called second",
			src: `package test
func TestAccExample(t *testing.T) {
	t.Parallel()
	testutil.PrintMetadata(t, testutil.AccTestMetadata{})
}`,
			expected: false,
		},
		{
			name: "Empty body",
			src: `package test
func TestAccExample(t *testing.T) {}`,
			expected: false,
		},
		{
			name: "Different function called first",
			src: `package test
func TestAccExample(t *testing.T) {
	otherpkg.PrintMetadata(t)
}`,
			expected: false,
		},
		{
			name: "Wrong function in testutil called first",
			src: `package test
func TestAccExample(t *testing.T) {
	testutil.OtherFunc(t)
}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := parseFuncDecl(t, tt.src)
			if actual := hasPrintMetadataFirst(fn, nil); actual != tt.expected {
				t.Errorf("hasPrintMetadataFirst() = %v, expected %v", actual, tt.expected)
			}
		})
	}
}
