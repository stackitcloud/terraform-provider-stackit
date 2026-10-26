package utils

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	serviceenablement "github.com/stackitcloud/stackit-sdk-go/services/serviceenablement/v2api"
)

func TestCheckServiceEnabled(t *testing.T) {
	type args struct {
		resp *serviceenablement.ServiceStatus
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "enabled",
			args: args{
				resp: &serviceenablement.ServiceStatus{
					State: new(serviceenablement.SERVICESTATUSSTATE_ENABLED),
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "enabling",
			args: args{
				resp: &serviceenablement.ServiceStatus{
					State: new(serviceenablement.SERVICESTATUSSTATE_ENABLING),
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "disabled",
			args: args{
				resp: &serviceenablement.ServiceStatus{
					State: new(serviceenablement.SERVICESTATUSSTATE_DISABLED),
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "disabling",
			args: args{
				resp: &serviceenablement.ServiceStatus{
					State: new(serviceenablement.SERVICESTATUSSTATE_DISABLING),
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "resp is nil",
			args: args{
				resp: nil,
			},
			wantErr: true,
		},
		{
			name: "resp state is nil",
			args: args{
				resp: &serviceenablement.ServiceStatus{
					State: nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckServiceEnabled(tt.args.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckServiceEnabled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("CheckServiceEnabled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateServiceIdAndRegionCombination(t *testing.T) {
	fixtureAvailableServicesByRegion := func() map[string][]string {
		return map[string][]string{
			"eu01": {
				"cloud.stackit.ske",
				"cloud.stackit.model-serving",
			},
			"eu02": {
				"cloud.stackit.ske",
			},
		}
	}

	type args struct {
		availableServicesByRegion map[string][]string
		serviceId                 string
		region                    string
	}
	tests := []struct {
		name string
		args args
		want *diag.ErrorDiagnostic
	}{
		{
			name: "SKE eu01",
			args: args{
				availableServicesByRegion: fixtureAvailableServicesByRegion(),
				serviceId:                 "cloud.stackit.ske",
				region:                    "eu01",
			},
			want: nil,
		},
		{
			name: "SKE eu02",
			args: args{
				availableServicesByRegion: fixtureAvailableServicesByRegion(),
				serviceId:                 "cloud.stackit.ske",
				region:                    "eu01",
			},
			want: nil,
		},
		{
			name: "Model serving eu02",
			args: args{
				availableServicesByRegion: fixtureAvailableServicesByRegion(),
				serviceId:                 "cloud.stackit.model-serving",
				region:                    "eu02",
			},
			want: new(diag.NewErrorDiagnostic("Invalid configuration", "The region 'eu02' is not available for the service 'cloud.stackit.model-serving'")),
		},
		{
			name: "SKE in non-existing region",
			args: args{
				availableServicesByRegion: map[string][]string{},
				serviceId:                 "cloud.stackit.ske",
				region:                    "foo-bar",
			},
			want: new(diag.NewErrorDiagnostic("Invalid configuration", "The region 'foo-bar' is not supported by the service enablement resource")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateServiceIdAndRegionCombination(tt.args.availableServicesByRegion, tt.args.serviceId, tt.args.region); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateServiceIdAndRegionCombination() = %v, want %v", got, tt.want)
			}
		})
	}
}
