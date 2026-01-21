package postgresflexalpha

import (
	"context"
	"testing"

	// The fwresource import alias is so there is no collision
	// with the more typical acceptance testing import:
	// "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestInstanceResourceSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	schemaRequest := fwresource.SchemaRequest{}
	schemaResponse := &fwresource.SchemaResponse{}

	// Instantiate the resource.Resource and call its Schema method
	NewInstanceResource().Schema(ctx, schemaRequest, schemaResponse)

	if schemaResponse.Diagnostics.HasError() {
		t.Fatalf("Schema method diagnostics: %+v", schemaResponse.Diagnostics)
	}

	// Validate the schema
	diagnostics := schemaResponse.Schema.ValidateImplementation(ctx)

	if diagnostics.HasError() {
		t.Fatalf("Schema validation diagnostics: %+v", diagnostics)
	}
}
