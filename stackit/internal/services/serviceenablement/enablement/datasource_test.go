package enablement

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
)

func Test_mapDatasourceFields(t *testing.T) {
	testProjectId := uuid.New().String()

	fixtureModel := func(mods ...func(model *DatasourceModel)) *DatasourceModel {
		model := &DatasourceModel{
			ResourceModel: ResourceModel{
				Id:        types.StringValue(fmt.Sprintf("%s,eu01,cloud.stackit.ske", testProjectId)),
				ProjectId: types.StringValue(testProjectId),
				Region:    types.StringValue("eu01"),
				ServiceId: types.StringValue("cloud.stackit.ske"),
			},
			Enabled: types.BoolNull(),
		}

		for _, mod := range mods {
			mod(model)
		}

		return model
	}

	fixtureResp := func(mods ...func(resp *serviceenablement.ServiceStatus)) *serviceenablement.ServiceStatus {
		status := &serviceenablement.ServiceStatus{
			State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
		}

		for _, mod := range mods {
			mod(status)
		}

		return status
	}

	type args struct {
		model  *DatasourceModel
		resp   *serviceenablement.ServiceStatus
		region string
	}
	tests := []struct {
		name    string
		args    args
		want    *DatasourceModel
		wantErr bool
	}{
		{
			name: "enabled",
			args: args{
				model:  fixtureModel(),
				resp:   fixtureResp(),
				region: "eu01",
			},
			want: fixtureModel(func(model *DatasourceModel) {
				model.Enabled = types.BoolValue(true)
			}),
		},
		{
			name: "enabling",
			args: args{
				model: fixtureModel(),
				resp: fixtureResp(func(resp *serviceenablement.ServiceStatus) {
					resp.State = new(serviceenablement.SERVICESTATUSSTATE_ENABLING)
				}),
				region: "eu01",
			},
			want: fixtureModel(func(model *DatasourceModel) {
				model.Enabled = types.BoolValue(true)
			}),
		},
		{
			name: "disabled",
			args: args{
				model: fixtureModel(),
				resp: fixtureResp(func(resp *serviceenablement.ServiceStatus) {
					resp.State = new(serviceenablement.SERVICESTATUSSTATE_DISABLED)
				}),
				region: "eu01",
			},
			want: fixtureModel(func(model *DatasourceModel) {
				model.Enabled = types.BoolValue(false)
			}),
		},
		{
			name: "disabling",
			args: args{
				model: fixtureModel(),
				resp: fixtureResp(func(resp *serviceenablement.ServiceStatus) {
					resp.State = new(serviceenablement.SERVICESTATUSSTATE_DISABLING)
				}),
				region: "eu01",
			},
			want: fixtureModel(func(model *DatasourceModel) {
				model.Enabled = types.BoolValue(false)
			}),
		},
		{
			name: "model is nil",
			args: args{
				model: nil,
				resp: &serviceenablement.ServiceStatus{
					State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
				},
			},
			wantErr: true,
		},
		{
			name: "resp is nil",
			args: args{
				model: &DatasourceModel{},
				resp:  nil,
			},
			wantErr: true,
		},
		{
			name: "resp state is nil",
			args: args{
				model: &DatasourceModel{},
				resp: &serviceenablement.ServiceStatus{
					State: nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapDatasourceFields(tt.args.model, tt.args.resp, tt.args.region); (err != nil) != tt.wantErr {
				t.Errorf("mapDatasourceFields() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if diff := cmp.Diff(tt.want, tt.args.model); diff != "" {
				t.Errorf("mapDatasourceFields() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
