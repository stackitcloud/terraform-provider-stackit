package tflogresponse

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	dir, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}

	analysistest.Run(t, dir, Analyzer, "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/tflogresponse")
}
