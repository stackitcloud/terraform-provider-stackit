package objectstorage

import (
	"context"
	_ "embed"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

type objectStorageClientMocked struct {
	returnError bool
}

func (c *objectStorageClientMocked) EnableServiceExecute(_ context.Context, projectId, region string) (*objectstorage.ProjectStatus, error) {
	if c.returnError {
		return nil, fmt.Errorf("create project failed")
	}

	return &objectstorage.ProjectStatus{
		Project: utils.Ptr(projectId),
	}, nil
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *objectstorage.GetBucketResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringNull(),
				URLVirtualHostedStyle: types.StringNull(),
				Region:                types.StringValue("eu01"),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{
					UrlPathStyle:          utils.Ptr("url/path/style"),
					UrlVirtualHostedStyle: utils.Ptr("url/virtual/hosted/style"),
				},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue("url/path/style"),
				URLVirtualHostedStyle: types.StringValue("url/virtual/hosted/style"),
				Region:                types.StringValue("eu01"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.GetBucketResponse{
				Bucket: &objectstorage.Bucket{
					UrlPathStyle:          utils.Ptr(""),
					UrlVirtualHostedStyle: utils.Ptr(""),
				},
			},
			Model{
				Id:                    types.StringValue("pid,bname"),
				Name:                  types.StringValue("bname"),
				ProjectId:             types.StringValue("pid"),
				URLPathStyle:          types.StringValue(""),
				URLVirtualHostedStyle: types.StringValue(""),
				Region:                types.StringValue("eu01"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"no_bucket",
			&objectstorage.GetBucketResponse{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
				Name:      tt.expected.Name,
			}
			err := mapFields(tt.input, model, "eu01")
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestEnableProject(t *testing.T) {
	tests := []struct {
		description string
		enableFails bool
		isValid     bool
	}{
		{
			"default_values",
			false,
			true,
		},
		{
			"error_response",
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			client := &objectStorageClientMocked{
				returnError: tt.enableFails,
			}
			err := enableProject(context.Background(), &Model{}, "eu01", client)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
		})
	}
}

func TestAdaptRegion(t *testing.T) {
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
			resp := resource.ModifyPlanResponse{
				Plan: tfsdk.Plan{
					Schema: schema.Schema{
						Attributes: map[string]schema.Attribute{
							"id":                       schema.StringAttribute{},
							"name":                     schema.StringAttribute{},
							"project_id":               schema.StringAttribute{},
							"url_path_style":           schema.StringAttribute{},
							"url_virtual_hosted_style": schema.StringAttribute{},
							"region":                   schema.StringAttribute{},
						},
					},
				},
			}
			configModel := Model{
				Region: tc.args.configRegion,
			}
			planModel := Model{}
			adaptRegion(context.Background(), &configModel, &planModel, tc.args.defaultRegion, &resp)
			if diags := resp.Diagnostics; tc.wantErr != diags.HasError() {
				t.Errorf("unexpected diagnostics: want err: %v, actual %v", tc.wantErr, diags.Errors())
			}
			if expected, actual := tc.wantRegion, planModel.Region; !expected.Equal(actual) {
				t.Errorf("wrong result region. expect %s but got %s", expected, actual)
			}
		})
	}

}
