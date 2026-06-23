package generic

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"
)

func Test_mapDatasourceFields(t *testing.T) {
	const testRegion = "eu01"
	testResourceId := uuid.New().String()

	type args struct {
		resp   []GenericRoleBindingResponse
		model  *DatasourceModel
		region string
	}
	tests := []struct {
		name      string
		args      args
		wantModel *DatasourceModel
		wantErr   bool
	}{
		{
			name: "default",
			args: args{
				resp: func() []GenericRoleBindingResponse {
					roleBinding1 := secretsmanagerV1Alpha.RoleBinding{
						Role:    "owner",
						Subject: "john.doe@example.com",
					}
					roleBinding2 := secretsmanagerV1Alpha.RoleBinding{
						Role:    "editor",
						Subject: "jane.doe@example.com",
					}

					return []GenericRoleBindingResponse{
						&roleBinding1,
						&roleBinding2,
					}
				}(),
				model: &DatasourceModel{
					ResourceId: types.StringValue(testResourceId),
				},
				region: testRegion,
			},
			wantErr: false,
			wantModel: &DatasourceModel{
				Id:         types.StringValue(fmt.Sprintf("%s,%s", testRegion, testResourceId)),
				ResourceId: types.StringValue(testResourceId),
				Region:     types.StringValue(testRegion),
				RoleBindings: []nestedRoleBinding{
					{
						Role:    types.StringValue("owner"),
						Subject: types.StringValue("john.doe@example.com"),
					},
					{
						Role:    types.StringValue("editor"),
						Subject: types.StringValue("jane.doe@example.com"),
					},
				},
			},
		},
		{
			name: "response is nil",
			args: args{
				resp: nil,
				model: &DatasourceModel{
					ResourceId: types.StringValue(testResourceId),
				},
				region: testRegion,
			},
			wantErr: false,
			wantModel: &DatasourceModel{
				Id:           types.StringValue(fmt.Sprintf("%s,%s", testRegion, testResourceId)),
				ResourceId:   types.StringValue(testResourceId),
				Region:       types.StringValue(testRegion),
				RoleBindings: []nestedRoleBinding{},
			},
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
				resp:  []GenericRoleBindingResponse{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapDatasourceFields(tt.args.resp, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapDatasourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			diff := cmp.Diff(tt.args.model, tt.wantModel)
			if diff != "" {
				t.Fatalf("Data does not match: %s", diff)
			}
		})
	}
}
