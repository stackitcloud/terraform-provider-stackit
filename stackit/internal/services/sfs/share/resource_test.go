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
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
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
		input    *sfs.Share
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
			&sfs.Share{
				Id:   testShareId.ValueStringPointer(),
				Name: new("testname"),
				ExportPolicy: *sfs.NewNullableShareExportPolicy(&sfs.ShareExportPolicy{
					Name: new("test-policy"),
				}),
				SpaceHardLimitGigabytes: utils.Ptr[int32](42),
			},
			&Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				ShareId:                 testShareId,
				Name:                    types.StringValue("testname"),
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int32Value(42),
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
			input: &sfs.Share{
				CreatedAt:               &testTime,
				Id:                      testShareId.ValueStringPointer(),
				MountPath:               new("mountpoint"),
				Name:                    new("testname"),
				SpaceHardLimitGigabytes: utils.Ptr[int32](42),
				State:                   new("state"),
				ExportPolicy: *sfs.NewNullableShareExportPolicy(&sfs.ShareExportPolicy{
					Name: new("test-policy"),
				}),
			},
			expected: &Model{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				Name:                    types.StringValue("testname"),
				ShareId:                 testShareId,
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int32Value(42),
				Region:                  testRegion,
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
				SpaceHardLimitGigabytes: types.Int32Value(42),
			},
			sfs.CreateSharePayload{
				ExportPolicyName:        *sfs.NewNullableString(new("test-policy")),
				Name:                    "testname",
				SpaceHardLimitGigabytes: 42,
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
				SpaceHardLimitGigabytes: types.Int32Value(42),
				ExportPolicyName:        testPolicyName,
			},
			&sfs.UpdateSharePayload{
				ExportPolicyName:        *sfs.NewNullableString(testPolicyName.ValueStringPointer()),
				SpaceHardLimitGigabytes: *sfs.NewNullableInt32(utils.Ptr[int32](42)),
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
				if diff := cmp.Diff(got, tt.want, cmp.AllowUnexported(sfs.NullableString{}, sfs.NullableInt32{})); diff != "" {
					t.Errorf("Data does not match: %s", diff)
				}
			}
		})
	}
}
