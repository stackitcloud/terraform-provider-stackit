package tfprintmetadata

import (
	"testing"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/testutil"
)

func TestAccValid(t *testing.T) {
	testutil.PrintMetadata(t, testutil.AccTestMetadata{
		ProductId: "test",
	})
}

func TestAccMissingPrintMetadata(t *testing.T) { // want "tfprintmetadata: test case TestAccMissingPrintMetadata must call testutil.PrintMetadata as the first statement"
	t.Log("doing something")
}

func TestAccEmpty(t *testing.T) { // want "tfprintmetadata: test case TestAccEmpty must call testutil.PrintMetadata as the first statement"
}

func TestAccNotFirst(t *testing.T) { // want "tfprintmetadata: test case TestAccNotFirst must call testutil.PrintMetadata as the first statement"
	t.Parallel()
	testutil.PrintMetadata(t, testutil.AccTestMetadata{
		ProductId: "test",
	})
}

func TestUnitNonAcc(t *testing.T) {
	t.Log("regular unit test without PrintMetadata")
}

type SomeType struct{}

func (s *SomeType) TestAccMethod(t *testing.T) {
	// Method with receiver should not be treated as a test case
}
