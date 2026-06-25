package testutils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"go.uber.org/mock/gomock"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance"
	mock_instance "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/instance/mock"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/token"
	mock_serviceenablement "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/modelexperiments/utils/mock"
)

type TestContext struct {
	T                           *testing.T
	MockCtrl                    *gomock.Controller
	MockInstanceCLient          *mock_instance.MockDefaultAPI
	MockServiceEnablementClient *mock_serviceenablement.MockDefaultAPI
	Ctx                         context.Context
}

func NewTestContext(t *testing.T) *TestContext {
	ctrl := gomock.NewController(t)
	mockClient := mock_instance.NewMockDefaultAPI(ctrl)
	mockServiceClient := mock_serviceenablement.NewMockDefaultAPI(ctrl)
	return &TestContext{
		T:                           t,
		MockCtrl:                    ctrl,
		MockInstanceCLient:          mockClient,
		MockServiceEnablementClient: mockServiceClient,
		Ctx:                         context.Background(),
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

func CreateInstanceRequest(ctx context.Context, schema resource.SchemaResponse, model instance.Model) resource.CreateRequest { //nolint:gocritic
	req := resource.CreateRequest{}
	req.Plan = tfsdk.Plan{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.Plan.Set(ctx, model)
	return req
}

func CreateInstanceTokenRequest(ctx context.Context, schema resource.SchemaResponse, model token.Model) resource.CreateRequest { //nolint:gocritic
	req := resource.CreateRequest{}
	req.Plan = tfsdk.Plan{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.Plan.Set(ctx, model)
	return req
}

func CreateResponse(schema resource.SchemaResponse) *resource.CreateResponse { //nolint:gocritic
	resp := &resource.CreateResponse{}
	resp.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	return resp
}

func UpdateInstanceRequest(ctx context.Context, schema resource.SchemaResponse, currentState, plannedState instance.Model) resource.UpdateRequest { //nolint:gocritic
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

func UpdateTokenRequest(ctx context.Context, schema resource.SchemaResponse, currentState, plannedState token.Model) resource.UpdateRequest { //nolint:gocritic
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

// UpdateInstanceResponse creates a test Update response
// Optionally initialize with current state to simulate Terraform framework behavior
func UpdateInstanceResponse(ctx context.Context, schema resource.SchemaResponse, currentState *instance.Model) *resource.UpdateResponse { //nolint:gocritic
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

func UpdateTokenResponse(ctx context.Context, schema resource.SchemaResponse, currentState *token.Model) *resource.UpdateResponse { //nolint:gocritic
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

// DeleteInstanceRequest creates a test Delete request
func DeleteInstanceRequest(ctx context.Context, schema resource.SchemaResponse, state instance.Model) resource.DeleteRequest { //nolint:gocritic
	req := resource.DeleteRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

func DeleteTokenRequest(ctx context.Context, schema resource.SchemaResponse, state token.Model) resource.DeleteRequest { //nolint:gocritic
	req := resource.DeleteRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

// DeleteInstanceResponse creates a test Delete response
// Optionally initialize with current state to simulate Terraform framework behavior
func DeleteInstanceResponse(ctx context.Context, schema resource.SchemaResponse, currentState *instance.Model) *resource.DeleteResponse { //nolint:gocritic
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

func DeleteTokenResponse(ctx context.Context, schema resource.SchemaResponse, currentState *token.Model) *resource.DeleteResponse { //nolint:gocritic
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

// ReadInstanceRequest creates a test Read request
func ReadInstanceRequest(ctx context.Context, schema resource.SchemaResponse, state instance.Model) resource.ReadRequest { //nolint:gocritic
	req := resource.ReadRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

func ReadTokenRequest(ctx context.Context, schema resource.SchemaResponse, state token.Model) resource.ReadRequest { //nolint:gocritic
	req := resource.ReadRequest{}
	req.State = tfsdk.State{
		Schema: schema.Schema,
		Raw:    tftypes.NewValue(tftypes.DynamicPseudoType, nil),
	}
	req.State.Set(ctx, state)
	return req
}

// ReadInstanceResponse creates a test Read response
func ReadInstanceResponse(ctx context.Context, schema resource.SchemaResponse, currentState *instance.Model) *resource.ReadResponse { //nolint:gocritic
	resp := &resource.ReadResponse{}
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

func ReadTokenResponse(ctx context.Context, schema resource.SchemaResponse, currentState *token.Model) *resource.ReadResponse { //nolint:gocritic
	resp := &resource.ReadResponse{}
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
