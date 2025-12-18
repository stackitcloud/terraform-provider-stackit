package share

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
)

var (
	testProjectId      = types.StringValue(uuid.NewString())
	testResourcePoolId = types.StringValue(uuid.NewString())
	testShareId        = types.StringValue(uuid.NewString())
	testRegion         = types.StringValue("eu01")
	testId             = types.StringValue(testProjectId.ValueString() + "," + testRegion.ValueString() + "," + testResourcePoolId.ValueString() + "," + testShareId.ValueString())
	testPolicyName     = types.StringValue("test-policy")
)

func TestMapFields(t *testing.T) {
	testTime := time.Now()
	tests := []struct {
		name     string
		state    *Model
		region   string
		input    *sfs.GetShareResponseShare
		expected *Model
		isValid  bool
	}{
		{
			"default_values",
			&Model{
				Id:             testId,
				ProjectId:      testProjectId,
				ResourcePoolId: testResourcePoolId,
			},
			"eu01",
			&sfs.GetShareResponseShare{
				Id:   testShareId.ValueStringPointer(),
				Name: utils.Ptr("testname"),
				ExportPolicy: sfs.NewNullableShareExportPolicy(&sfs.ShareExportPolicy{
					Name: utils.Ptr("test-policy"),
				}),
				SpaceHardLimitGigabytes: utils.Ptr[int64](42),
			},
			&Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				ShareId:                 testShareId,
				Name:                    types.StringValue("testname"),
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int64Value(42),
				Region:                  types.StringValue("eu01"),
			},
			true,
		},
		{
			name: "simple_values",
			state: &Model{
				Id:             testId,
				ProjectId:      testProjectId,
				ResourcePoolId: testResourcePoolId,
			},
			region: "eu01",
			input: &sfs.GetShareResponseShare{
				CreatedAt:               &testTime,
				Id:                      testShareId.ValueStringPointer(),
				MountPath:               utils.Ptr("mountpoint"),
				Name:                    utils.Ptr("testname"),
				SpaceHardLimitGigabytes: sfs.PtrInt64(42),
				State:                   utils.Ptr("state"),
				ExportPolicy: sfs.NewNullableShareExportPolicy(&sfs.ShareExportPolicy{
					Name: utils.Ptr("test-policy"),
				}),
			},
			expected: &Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				Name:                    types.StringValue("testname"),
				ShareId:                 testShareId,
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int64Value(42),
				Region:                  types.StringValue("eu01"),
				MountPath:               types.StringValue("mountpoint"),
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.input, tt.region, tt.state); (err == nil) != tt.isValid {
				t.Errorf("unexpected error")
			}
			if tt.isValid {
				if diff := cmp.Diff(tt.state, tt.expected); diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    sfs.CreateSharePayload
		wantErr bool
	}{
		{
			"default",
			&Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				ShareId:                 testShareId,
				Name:                    types.StringValue("testname"),
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int64Value(42),
			},
			sfs.CreateSharePayload{
				ExportPolicyName:        sfs.NewNullableString(utils.Ptr("test-policy")),
				Name:                    sfs.PtrString("testname"),
				SpaceHardLimitGigabytes: sfs.PtrInt64(42),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toCreatePayload(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(got, tt.want, cmp.AllowUnexported(sfs.NullableString{})); diff != "" {
					t.Errorf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    *sfs.UpdateSharePayload
		wantErr bool
	}{
		{
			"default",
			&Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				ShareId:                 testShareId,
				Name:                    types.StringValue("testname"),
				SpaceHardLimitGigabytes: types.Int64Value(42),
				ExportPolicyName:        testPolicyName,
			},
			&sfs.UpdateSharePayload{
				ExportPolicyName:        sfs.NewNullableString(testPolicyName.ValueStringPointer()),
				SpaceHardLimitGigabytes: sfs.PtrInt64(42),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUpdatePayload(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(got, tt.want, cmp.AllowUnexported(sfs.NullableString{})); diff != "" {
					t.Errorf("Data does not match: %s", diff)
				}
			}
		})
	}
}
