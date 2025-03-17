package features

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestValidExperiment(t *testing.T) {
	type args struct {
		experiment string
		diags      *diag.Diagnostics
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid",
			args: args{
				experiment: "iam",
				diags:      &diag.Diagnostics{},
			},
			want: true,
		},
		{
			name: "invalid",
			args: args{
				experiment: "foo",
				diags:      &diag.Diagnostics{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidExperiment(tt.args.experiment, tt.args.diags); got != tt.want {
				t.Errorf("ValidExperiment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckExperimentEnabled(t *testing.T) {
	type args struct {
		ctx          context.Context
		data         *core.ProviderData
		experiment   string
		resourceType string
		diags        *diag.Diagnostics
	}
	tests := []struct {
		name             string
		args             args
		wantDiagsErr     bool
		wantDiagsWarning bool
	}{
		{
			name: "enabled",
			args: args{
				ctx: context.Background(),
				data: &core.ProviderData{
					Experiments: []string{"iam"},
				},
				experiment: "iam",
				diags:      &diag.Diagnostics{},
			},
			wantDiagsErr:     false,
			wantDiagsWarning: true,
		},
		{
			name: "disabled",
			args: args{
				ctx: context.Background(),
				data: &core.ProviderData{
					Experiments: []string{},
				},
				experiment: "iam",
				diags:      &diag.Diagnostics{},
			},
			wantDiagsErr:     true,
			wantDiagsWarning: false,
		},
		{
			name: "invalid experiment",
			args: args{
				ctx: context.Background(),
				data: &core.ProviderData{
					Experiments: []string{"iam"},
				},
				experiment:   "foobar",
				resourceType: "provider",
				diags:        &diag.Diagnostics{},
			},
			wantDiagsErr:     true,
			wantDiagsWarning: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CheckExperimentEnabled(tt.args.ctx, tt.args.data, tt.args.experiment, tt.args.resourceType, tt.args.diags)
			if got := tt.args.diags.HasError(); got != tt.wantDiagsErr {
				t.Errorf("CheckExperimentEnabled() diags.HasError() = %v, want %v", got, tt.wantDiagsErr)
			}
			if got := tt.args.diags.WarningsCount() > 0; got != tt.wantDiagsWarning {
				t.Errorf("CheckExperimentEnabled() diags.WarningsCount() > 0 = %v, want %v", got, tt.wantDiagsErr)
			}
		})
	}
}
