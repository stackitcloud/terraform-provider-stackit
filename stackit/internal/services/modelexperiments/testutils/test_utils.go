package testutils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	mock_instance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance/mock"
	"go.uber.org/mock/gomock"
)

type TestContext struct {
	T                  *testing.T
	MockCtrl           *gomock.Controller
	MockInstanceCLient *mock_instance.MockDefaultAPI
	Ctx                context.Context
}

func NewTestContext(t *testing.T) *TestContext {
	ctrl := gomock.NewController(t)
	mockClient := mock_instance.NewMockDefaultAPI(ctrl)
	return &TestContext{
		T:                  t,
		MockCtrl:           ctrl,
		MockInstanceCLient: mockClient,
		Ctx:                context.Background(),
	}
}

func CreateInstanceTestModel(projectId, region, name, description string) instance.Model {
	return instance.Model{
		ProjectId:   types.StringValue(projectId),
		Region:      types.StringValue(region),
		Name:        types.StringValue(name),
		Description: types.StringValue(description),
		Labels:      types.MapNull(types.StringType),
	}
}

func CreateInstanceRequest(ctx context.Context, schema resource.SchemaResponse, model instance.Model) resource.CreateRequest {
	req := resource.CreateRequest{}
	req.Plan = tfsdk.Plan{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.Plan.Set(ctx, model)
	return req
}

func CreateResponse(schema resource.SchemaResponse) *resource.CreateResponse {
	resp := &resource.CreateResponse{}
	resp.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	return resp
}

func UpdateRequest(ctx context.Context, schema resource.SchemaResponse, currentState, plannedState instance.Model) resource.UpdateRequest {
	req := resource.UpdateRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.Plan = tfsdk.Plan{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, currentState)
	req.Plan.Set(ctx, plannedState)
	return req
}

// UpdateResponse creates a test Update response
// Optionally initialize with current state to simulate Terraform framework behavior
func UpdateResponse(ctx context.Context, schema resource.SchemaResponse, currentState *instance.Model) *resource.UpdateResponse {
	resp := &resource.UpdateResponse{}
	resp.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	// Initialize with current state to simulate framework behavior
	// When Update errors without calling State.Set(), this state is preserved
	if currentState != nil {
		resp.State.Set(ctx, *currentState)
	}
	return resp
}

// DeleteRequest creates a test Delete request
func DeleteRequest(ctx context.Context, schema resource.SchemaResponse, state instance.Model) resource.DeleteRequest {
	req := resource.DeleteRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

// DeleteResponse creates a test Delete response
// Optionally initialize with current state to simulate Terraform framework behavior
func DeleteResponse(ctx context.Context, schema resource.SchemaResponse, currentState *instance.Model) *resource.DeleteResponse {
	resp := &resource.DeleteResponse{}
	resp.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	// Initialize with current state to simulate framework behavior
	// When Delete errors without calling State.RemoveResource(), this state is preserved
	if currentState != nil {
		resp.State.Set(ctx, *currentState)
	}
	return resp
}

// ReadRequest creates a test Read request
func ReadRequest(ctx context.Context, schema resource.SchemaResponse, state instance.Model) resource.ReadRequest {
	req := resource.ReadRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

// ReadResponse creates a test Read response
func ReadResponse(schema resource.SchemaResponse) *resource.ReadResponse {
	resp := &resource.ReadResponse{}
	resp.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	return resp
}
