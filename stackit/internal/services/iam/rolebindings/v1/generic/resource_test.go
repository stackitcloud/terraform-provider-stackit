package generic

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	secretsmanagerV1Alpha "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1alphaapi"
)

func Test_mapFields(t *testing.T) {
	const testRegion = "eu01"
	resourceId := uuid.New().String()

	type args struct {
		resp   GenericRoleBindingResponse
		model  *Model
		region string
	}
	tests := []struct {
		name      string
		args      args
		wantModel *Model
		wantErr   bool
	}{
		{
			name: "default",
			args: args{
				region: testRegion,
				resp: &secretsmanagerV1Alpha.RoleBinding{
					Role:    "owner",
					Subject: "john.doe@example.com",
				},
				model: &Model{
					ResourceId: types.StringValue(resourceId),
				},
			},
			wantModel: &Model{
				Id:         types.StringValue(fmt.Sprintf("%s,%s,owner,john.doe@example.com", testRegion, resourceId)),
				ResourceId: types.StringValue(resourceId),
				Role:       types.StringValue("owner"),
				Subject:    types.StringValue("john.doe@example.com"),
				Region:     types.StringValue(testRegion),
			},
			wantErr: false,
		},
		{
			name: "model is nil",
			args: args{
				resp:  &secretsmanagerV1Alpha.RoleBinding{},
				model: nil,
			},
			wantErr: true,
		},
		{
			name: "response is nil",
			args: args{
				resp:  nil,
				model: &Model{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapFields(tt.args.resp, tt.args.model, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapFields() error = %v, wantErr %v", err, tt.wantErr)
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
