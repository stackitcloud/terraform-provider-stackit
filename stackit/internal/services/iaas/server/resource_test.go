package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas/wait"
)

const (
	userData              = "user_data"
	base64EncodedUserData = "dXNlcl9kYXRh"
	testTimestampValue    = "2006-01-02T15:04:05Z"
)

func testTimestamp() time.Time {
	timestamp, _ := time.Parse(time.RFC3339, testTimestampValue)
	return timestamp
}

func TestMapFields(t *testing.T) {
	type args struct {
		state  Model
		input  *iaas.Server
		region string
	}
	tests := []struct {
		description string
		args        args
		expected    Model
		isValid     bool
	}{
		{
			description: "default_values",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
				},
				input: &iaas.Server{
					Id: utils.Ptr("sid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                types.StringValue("pid,eu01,sid"),
				ProjectId:         types.StringValue("pid"),
				ServerId:          types.StringValue("sid"),
				Name:              types.StringNull(),
				AvailabilityZone:  types.StringNull(),
				Labels:            types.MapNull(types.StringType),
				ImageId:           types.StringNull(),
				NetworkInterfaces: types.ListNull(types.StringType),
				KeypairName:       types.StringNull(),
				Agent:             types.ObjectNull(agentTypes),
				AffinityGroup:     types.StringNull(),
				UserData:          types.StringNull(),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				LaunchedAt:        types.StringNull(),
				Region:            types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "simple_values",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
					Region:    types.StringValue("eu01"),
				},
				input: &iaas.Server{
					Id:               utils.Ptr("sid"),
					Name:             utils.Ptr("name"),
					AvailabilityZone: utils.Ptr("zone"),
					Labels: &map[string]interface{}{
						"key": "value",
					},
					ImageId: utils.Ptr("image_id"),
					Nics: &[]iaas.ServerNetwork{
						{
							NicId: utils.Ptr("nic1"),
						},
						{
							NicId: utils.Ptr("nic2"),
						},
					},
					KeypairName: utils.Ptr("keypair_name"),
					Agent: &iaas.ServerAgent{
						Provisioned: utils.Ptr(true),
					},
					AffinityGroup: utils.Ptr("group_id"),
					CreatedAt:     utils.Ptr(testTimestamp()),
					UpdatedAt:     utils.Ptr(testTimestamp()),
					LaunchedAt:    utils.Ptr(testTimestamp()),
					Status:        utils.Ptr("active"),
				},
				region: "eu02",
			},
			expected: Model{
				Id:               types.StringValue("pid,eu02,sid"),
				ProjectId:        types.StringValue("pid"),
				ServerId:         types.StringValue("sid"),
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				ImageId:           types.StringValue("image_id"),
				NetworkInterfaces: types.ListNull(types.StringType),
				KeypairName:       types.StringValue("keypair_name"),
				Agent: types.ObjectValueMust(agentTypes, map[string]attr.Value{
					"provisioned": types.BoolValue(true),
				}),
				AffinityGroup: types.StringValue("group_id"),
				CreatedAt:     types.StringValue(testTimestampValue),
				UpdatedAt:     types.StringValue(testTimestampValue),
				LaunchedAt:    types.StringValue(testTimestampValue),
				Region:        types.StringValue("eu02"),
			},
			isValid: true,
		},
		{
			description: "empty_labels",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
					ServerId:  types.StringValue("sid"),
					Labels:    types.MapValueMust(types.StringType, map[string]attr.Value{}),
				},
				input: &iaas.Server{
					Id: utils.Ptr("sid"),
				},
				region: "eu01",
			},
			expected: Model{
				Id:                types.StringValue("pid,eu01,sid"),
				ProjectId:         types.StringValue("pid"),
				ServerId:          types.StringValue("sid"),
				Name:              types.StringNull(),
				AvailabilityZone:  types.StringNull(),
				Labels:            types.MapValueMust(types.StringType, map[string]attr.Value{}),
				ImageId:           types.StringNull(),
				NetworkInterfaces: types.ListNull(types.StringType),
				KeypairName:       types.StringNull(),
				Agent:             types.ObjectNull(agentTypes),
				AffinityGroup:     types.StringNull(),
				UserData:          types.StringNull(),
				CreatedAt:         types.StringNull(),
				UpdatedAt:         types.StringNull(),
				LaunchedAt:        types.StringNull(),
				Region:            types.StringValue("eu01"),
			},
			isValid: true,
		},
		{
			description: "response_nil_fail",
		},
		{
			description: "no_resource_id",
			args: args{
				state: Model{
					ProjectId: types.StringValue("pid"),
				},
				input: &iaas.Server{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.args.input, &tt.args.state, tt.args.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.args.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.CreateServerPayload
		isValid     bool
	}{
		{
			description: "ok",
			input: &Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				BootVolume: types.ObjectValueMust(bootVolumeTypes, map[string]attr.Value{
					"performance_class":     types.StringValue("class"),
					"size":                  types.Int64Value(1),
					"source_type":           types.StringValue("type"),
					"source_id":             types.StringValue("id"),
					"delete_on_termination": types.BoolUnknown(),
					"id":                    types.StringValue("id"),
				}),
				ImageId:     types.StringValue("image"),
				KeypairName: types.StringValue("keypair"),
				MachineType: types.StringValue("machine_type"),
				UserData:    types.StringValue(userData),
				NetworkInterfaces: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("nic1"),
					types.StringValue("nic2"),
				}),
				Agent: types.ObjectValueMust(agentTypes, map[string]attr.Value{
					"provisioned": types.BoolValue(true),
				}),
			},
			expected: &iaas.CreateServerPayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				BootVolume: &iaas.ServerBootVolume{
					PerformanceClass: utils.Ptr("class"),
					Size:             utils.Ptr(int64(1)),
					Source: &iaas.BootVolumeSource{
						Type: utils.Ptr("type"),
						Id:   utils.Ptr("id"),
					},
				},
				ImageId:     utils.Ptr("image"),
				KeypairName: utils.Ptr("keypair"),
				MachineType: utils.Ptr("machine_type"),
				UserData:    utils.Ptr([]byte(base64EncodedUserData)),
				Networking: &iaas.CreateServerPayloadAllOfNetworking{
					CreateServerNetworkingWithNics: &iaas.CreateServerNetworkingWithNics{
						NicIds: &[]string{"nic1", "nic2"},
					},
				},
				Agent: &iaas.ServerAgent{
					Provisioned: utils.Ptr(true),
				},
			},
			isValid: true,
		},
		{
			description: "delete on termination is set to true",
			input: &Model{
				Name:             types.StringValue("name"),
				AvailabilityZone: types.StringValue("zone"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
				BootVolume: types.ObjectValueMust(bootVolumeTypes, map[string]attr.Value{
					"performance_class":     types.StringValue("class"),
					"size":                  types.Int64Value(1),
					"source_type":           types.StringValue("image"),
					"source_id":             types.StringValue("id"),
					"delete_on_termination": types.BoolValue(true),
					"id":                    types.StringValue("id"),
				}),
				ImageId:     types.StringValue("image"),
				KeypairName: types.StringValue("keypair"),
				MachineType: types.StringValue("machine_type"),
				UserData:    types.StringValue(userData),
				NetworkInterfaces: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("nic1"),
					types.StringValue("nic2"),
				}),
				Agent: types.ObjectValueMust(agentTypes, map[string]attr.Value{
					"provisioned": types.BoolValue(true),
				}),
			},
			expected: &iaas.CreateServerPayload{
				Name:             utils.Ptr("name"),
				AvailabilityZone: utils.Ptr("zone"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
				BootVolume: &iaas.ServerBootVolume{
					PerformanceClass: utils.Ptr("class"),
					Size:             utils.Ptr(int64(1)),
					Source: &iaas.BootVolumeSource{
						Type: utils.Ptr("image"),
						Id:   utils.Ptr("id"),
					},
					DeleteOnTermination: utils.Ptr(true),
				},
				ImageId:     utils.Ptr("image"),
				KeypairName: utils.Ptr("keypair"),
				MachineType: utils.Ptr("machine_type"),
				UserData:    utils.Ptr([]byte(base64EncodedUserData)),
				Networking: &iaas.CreateServerPayloadAllOfNetworking{
					CreateServerNetworkingWithNics: &iaas.CreateServerNetworkingWithNics{
						NicIds: &[]string{"nic1", "nic2"},
					},
				},
				Agent: &iaas.ServerAgent{
					Provisioned: utils.Ptr(true),
				},
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *iaas.UpdateServerPayload
		isValid     bool
	}{
		{
			"default_ok",
			&Model{
				Name: types.StringValue("name"),
				Labels: types.MapValueMust(types.StringType, map[string]attr.Value{
					"key": types.StringValue("value"),
				}),
			},
			&iaas.UpdateServerPayload{
				Name: utils.Ptr("name"),
				Labels: &map[string]interface{}{
					"key": "value",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toUpdatePayload(context.Background(), tt.input, types.MapNull(types.StringType))
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

var _ serverControlClient = &mockServerControlClient{}

// mockServerControlClient mocks the [serverControlClient] interface with
// pluggable functions
type mockServerControlClient struct {
	wait.APIClientInterface
	startServerCalled  int
	startServerExecute func(callNo int, ctx context.Context, projectId, region, serverId string) error

	stopServerCalled  int
	stopServerExecute func(callNo int, ctx context.Context, projectId, region, serverId string) error

	deallocateServerCalled  int
	deallocateServerExecute func(callNo int, ctx context.Context, projectId, region, serverId string) error

	getServerCalled  int
	getServerExecute func(callNo int, ctx context.Context, projectId, region, serverId string) (*iaas.Server, error)
}

// DeallocateServerExecute implements serverControlClient.
func (t *mockServerControlClient) DeallocateServerExecute(ctx context.Context, projectId, region, serverId string) error {
	t.deallocateServerCalled++
	return t.deallocateServerExecute(t.deallocateServerCalled, ctx, projectId, region, serverId)
}

// GetServerExecute implements serverControlClient.
func (t *mockServerControlClient) GetServerExecute(ctx context.Context, projectId, region, serverId string) (*iaas.Server, error) {
	t.getServerCalled++
	return t.getServerExecute(t.getServerCalled, ctx, projectId, region, serverId)
}

// StartServerExecute implements serverControlClient.
func (t *mockServerControlClient) StartServerExecute(ctx context.Context, projectId, region, serverId string) error {
	t.startServerCalled++
	return t.startServerExecute(t.startServerCalled, ctx, projectId, region, serverId)
}

// StopServerExecute implements serverControlClient.
func (t *mockServerControlClient) StopServerExecute(ctx context.Context, projectId, region, serverId string) error {
	t.stopServerCalled++
	return t.stopServerExecute(t.stopServerCalled, ctx, projectId, region, serverId)
}

func Test_serverResource_updateServerStatus(t *testing.T) {
	projectId := basetypes.NewStringValue("projectId")
	serverId := basetypes.NewStringValue("serverId")
	type fields struct {
		client *mockServerControlClient
	}
	type args struct {
		currentState *string
		model        Model
		region       string
	}
	type want struct {
		err              bool
		status           types.String
		getServerCount   int
		stopCount        int
		startCount       int
		deallocatedCount int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "no desired status",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(_ int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: utils.Ptr(wait.ServerActiveStatus),
						}, nil
					},
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerActiveStatus),
				model: Model{
					ProjectId: projectId,
					ServerId:  serverId,
				},
			},
			want: want{
				getServerCount: 1,
			},
		},

		{
			name: "desired inactive state",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(no int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						var state string
						if no <= 1 {
							state = wait.ServerActiveStatus
						} else {
							state = wait.ServerInactiveStatus
						}
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: &state,
						}, nil
					},
					stopServerExecute: func(_ int, _ context.Context, _, _, _ string) error { return nil },
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerActiveStatus),
				model: Model{
					ProjectId:     projectId,
					ServerId:      serverId,
					DesiredStatus: basetypes.NewStringValue("inactive"),
				},
			},
			want: want{
				getServerCount: 2,
				stopCount:      1,
				status:         basetypes.NewStringValue("inactive"),
			},
		},
		{
			name: "desired deallocated state",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(no int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						var state string
						switch no {
						case 1:
							state = wait.ServerActiveStatus
						case 2:
							state = wait.ServerInactiveStatus
						default:
							state = wait.ServerDeallocatedStatus
						}
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: &state,
						}, nil
					},
					deallocateServerExecute: func(_ int, _ context.Context, _, _, _ string) error { return nil },
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerActiveStatus),
				model: Model{
					ProjectId:     projectId,
					ServerId:      serverId,
					DesiredStatus: basetypes.NewStringValue("deallocated"),
				},
			},
			want: want{
				getServerCount:   3,
				deallocatedCount: 1,
				status:           basetypes.NewStringValue("deallocated"),
			},
		},
		{
			name: "don't call start if active",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(_ int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: utils.Ptr(wait.ServerActiveStatus),
						}, nil
					},
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerActiveStatus),
				model: Model{
					ProjectId:     projectId,
					ServerId:      serverId,
					DesiredStatus: basetypes.NewStringValue("active"),
				},
			},
			want: want{
				status:         basetypes.NewStringValue("active"),
				getServerCount: 1,
			},
		},
		{
			name: "don't call stop if inactive",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(_ int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: utils.Ptr(wait.ServerInactiveStatus),
						}, nil
					},
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerInactiveStatus),
				model: Model{
					ProjectId:     projectId,
					ServerId:      serverId,
					DesiredStatus: basetypes.NewStringValue("inactive"),
				},
			},
			want: want{
				status:         basetypes.NewStringValue("inactive"),
				getServerCount: 1,
			},
		},
		{
			name: "don't call dealloacate if deallocated",
			fields: fields{
				client: &mockServerControlClient{
					getServerExecute: func(_ int, _ context.Context, _, _, _ string) (*iaas.Server, error) {
						return &iaas.Server{
							Id:     utils.Ptr(serverId.ValueString()),
							Status: utils.Ptr(wait.ServerDeallocatedStatus),
						}, nil
					},
				},
			},
			args: args{
				currentState: utils.Ptr(wait.ServerDeallocatedStatus),
				model: Model{
					ProjectId:     projectId,
					ServerId:      serverId,
					DesiredStatus: basetypes.NewStringValue("deallocated"),
				},
			},
			want: want{
				status:         basetypes.NewStringValue("deallocated"),
				getServerCount: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := updateServerStatus(context.Background(), tt.fields.client, tt.args.currentState, &tt.args.model, tt.args.region)
			if (err != nil) != tt.want.err {
				t.Errorf("inconsistent error, want %v and got %v", tt.want.err, err)
			}
			if expected, actual := tt.want.status, tt.args.model.DesiredStatus; expected != actual {
				t.Errorf("wanted status %s but got %s", expected, actual)
			}

			if expected, actual := tt.want.getServerCount, tt.fields.client.getServerCalled; expected != actual {
				t.Errorf("wrong number of get server calls: Expected %d but got %d", expected, actual)
			}
			if expected, actual := tt.want.startCount, tt.fields.client.startServerCalled; expected != actual {
				t.Errorf("wrong number of start server calls: Expected %d but got %d", expected, actual)
			}
			if expected, actual := tt.want.stopCount, tt.fields.client.stopServerCalled; expected != actual {
				t.Errorf("wrong number of stop server calls: Expected %d but got %d", expected, actual)
			}
			if expected, actual := tt.want.deallocatedCount, tt.fields.client.deallocateServerCalled; expected != actual {
				t.Errorf("wrong number of deallocate server calls: Expected %d but got %d", expected, actual)
			}
		})
	}
}
