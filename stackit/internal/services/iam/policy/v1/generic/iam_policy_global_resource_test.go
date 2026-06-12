package generic

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	resourcemanagerV1Beta "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v1betaapi"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
)

func Test_mapFieldsGlobal(t *testing.T) {
	const testRegion = "eu01"
	resourceId := uuid.New().String()

	type args struct {
		resp       *IAMPolicy
		inputModel *GlobalModel
		region     string
	}
	tests := []struct {
		name      string
		args      args
		wantModel *GlobalModel
		wantErr   bool
	}{
		{
			name: "default",
			args: args{
				resp: &IAMPolicy{
					Etag: new("etag-value"),
					RoleBindings: utils.Map([]resourcemanagerV1Beta.RoleBinding{
						{
							Subject: "jane.doe@example.com",
							Role:    "owner",
						},
						{
							Subject: "john.doe@example.com",
							Role:    "editor",
						},
					}, func(t resourcemanagerV1Beta.RoleBinding) RoleBinding {
						return &t
					}),
				},
				inputModel: &GlobalModel{
					ResourceId: types.StringValue(resourceId),
				},
				region: testRegion,
			},
			wantModel: &GlobalModel{
				ResourceId: types.StringValue(resourceId),
				Id:         types.StringValue(resourceId),
				Etag:       types.StringValue("etag-value"),
				RoleBindings: []roleBindingModel{
					{
						Subject: types.StringValue("jane.doe@example.com"),
						Role:    types.StringValue("owner"),
					},
					{
						Subject: types.StringValue("john.doe@example.com"),
						Role:    types.StringValue("editor"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "role binding slice is empty in API response",
			args: args{
				resp: &IAMPolicy{
					Etag: nil,
					RoleBindings: utils.Map([]resourcemanagerV1Beta.RoleBinding{}, func(t resourcemanagerV1Beta.RoleBinding) RoleBinding {
						return &t
					}),
				},
				inputModel: &GlobalModel{
					ResourceId: types.StringValue(resourceId),
				},
				region: testRegion,
			},
			wantModel: &GlobalModel{
				ResourceId:   types.StringValue(resourceId),
				Id:           types.StringValue(resourceId),
				Etag:         types.StringNull(),
				RoleBindings: []roleBindingModel{},
			},
			wantErr: false,
		},
		{
			name: "role binding slice is nil in API response",
			args: args{
				resp: &IAMPolicy{
					Etag:         new("etag-value"),
					RoleBindings: nil,
				},
				inputModel: &GlobalModel{
					ResourceId: types.StringValue(resourceId),
				},
				region: testRegion,
			},
			wantModel: &GlobalModel{
				ResourceId:   types.StringValue(resourceId),
				Id:           types.StringValue(resourceId),
				Etag:         types.StringValue("etag-value"),
				RoleBindings: []roleBindingModel{},
			},
			wantErr: false,
		},
		{
			name: "response is nil",
			args: args{
				resp:       nil,
				inputModel: &GlobalModel{},
			},
			wantErr: true,
		},
		{
			name: "model is nil",
			args: args{
				resp:       &IAMPolicy{},
				inputModel: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapFieldsGlobal(tt.args.resp, tt.args.inputModel); (err != nil) != tt.wantErr {
				t.Errorf("mapFieldsGlobal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			diff := cmp.Diff(tt.args.inputModel, tt.wantModel)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}

func TestGlobalIAMPolicyResource_GetResourceName(t *testing.T) {
	tests := []struct {
		name     string
		resource GlobalIAMPolicyResource[resourcemanagerV1Beta.DefaultAPI]
		want     string
	}{
		{
			name: "default",
			resource: GlobalIAMPolicyResource[resourcemanagerV1Beta.DefaultAPI]{
				ApiName:      "resourcemanager",
				ResourceType: "folder",
			},
			want: "stackit_resourcemanager_folder_iam_policy_v1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resource.GetResourceName(); got != tt.want {
				t.Errorf("GetResourceName() = %v, want %v", got, tt.want)
			}
		})
	}
}
