package enablement

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Test_mapFields(t *testing.T) {
	testProjectId := uuid.New().String()

	type args struct {
		model  *ResourceModel
		region string
	}
	tests := []struct {
		name    string
		args    args
		want    *ResourceModel
		wantErr bool
	}{
		{
			name: "default",
			args: args{
				model: &ResourceModel{
					ProjectId: types.StringValue(testProjectId),
					ServiceId: types.StringValue("cloud.stackit.ske"),
					Region:    types.StringValue("eu02"),
				},
				region: "eu01",
			},
			want: &ResourceModel{
				ProjectId: types.StringValue(testProjectId),
				ServiceId: types.StringValue("cloud.stackit.ske"),
				Region:    types.StringValue("eu01"),
				Id:        types.StringValue(fmt.Sprintf("%s,eu01,cloud.stackit.ske", testProjectId)),
			},
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapFields(tt.args.model, nil, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapFields() error = %v, wantErr %v", err, tt.wantErr)
			}

			if diff := cmp.Diff(tt.want, tt.args.model); diff != "" {
				t.Errorf("mapFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
