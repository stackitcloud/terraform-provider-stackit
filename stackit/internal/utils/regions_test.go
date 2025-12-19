// Copyright (c) STACKIT

package utils

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAdaptRegion(t *testing.T) {
	type model struct {
		Region types.String `tfsdk:"region"`
	}
	type args struct {
		configRegion  types.String
		defaultRegion string
	}
	testcases := []struct {
		name       string
		args       args
		wantErr    bool
		wantRegion types.String
	}{
		{
			"no configured region, use provider region",
			args{
				types.StringNull(),
				"eu01",
			},
			false,
			types.StringValue("eu01"),
		},
		{
			"no configured region, no provider region => want error",
			args{
				types.StringNull(),
				"",
			},
			true,
			types.StringNull(),
		},
		{
			"configuration region overrides provider region",
			args{
				types.StringValue("eu01-m"),
				"eu01",
			},
			false,
			types.StringValue("eu01-m"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			plan := tfsdk.Plan{
				Schema: schema.Schema{
					Attributes: map[string]schema.Attribute{
						"region": schema.StringAttribute{
							Required: true,
						},
					},
				},
			}

			if diags := plan.Set(context.Background(), model{types.StringValue("unknown")}); diags.HasError() {
				t.Fatalf("cannot create test model: %v", diags)
			}
			resp := resource.ModifyPlanResponse{
				Plan: plan,
			}

			configModel := model{
				Region: tc.args.configRegion,
			}
			planModel := model{}
			AdaptRegion(context.Background(), configModel.Region, &planModel.Region, tc.args.defaultRegion, &resp)
			if diags := resp.Diagnostics; tc.wantErr != diags.HasError() {
				t.Errorf("unexpected diagnostics: want err: %v, actual %v", tc.wantErr, diags.Errors())
			}
			if expected, actual := tc.wantRegion, planModel.Region; !expected.Equal(actual) {
				t.Errorf("wrong result region. expect %s but got %s", expected, actual)
			}
		})
	}
}
