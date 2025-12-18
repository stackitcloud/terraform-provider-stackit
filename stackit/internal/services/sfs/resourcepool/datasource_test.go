package resourcepool

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
)

func TestMapDatasourceFields(t *testing.T) {
	now := time.Now()
	testTime := types.StringValue(now.Format(time.RFC3339))
	testTimePlus1h := types.StringValue(now.Add(1 * time.Hour).Format(time.RFC3339))
	tests := []struct {
		name     string
		state    *dataSourceModel
		region   string
		input    *sfs.GetResourcePoolResponseResourcePool
		expected *dataSourceModel
		isValid  bool
	}{
		{
			"default_values",
			&dataSourceModel{
				Id:        testId,
				ProjectId: testProjectId,
			},
			"eu01",
			&sfs.GetResourcePoolResponseResourcePool{
				Id: testResourcePoolId.ValueStringPointer(),
			},
			&dataSourceModel{
				Id:                             testId,
				ProjectId:                      testProjectId,
				ResourcePoolId:                 testResourcePoolId,
				AvailabilityZone:               types.StringNull(),
				IpAcl:                          types.ListNull(types.StringType),
				Name:                           types.StringNull(),
				PerformanceClass:               types.StringNull(),
				SizeGigabytes:                  types.Int64Null(),
				Region:                         testRegion,
				SizeReducibleAt:                types.StringNull(),
				PerformanceClassDowngradableAt: types.StringNull(),
			},
			true,
		},
		{
			name: "simple_values",
			state: &dataSourceModel{
				Id:        testId,
				ProjectId: testProjectId,
			},
			region: "eu01",
			input: &sfs.GetResourcePoolResponseResourcePool{
				AvailabilityZone: testAvailabilityZone.ValueStringPointer(),
				CountShares:      utils.Ptr[int64](42),
				CreatedAt:        &now,
				Id:               testResourcePoolId.ValueStringPointer(),
				IpAcl:            &[]string{"foo", "bar", "baz"},
				MountPath:        utils.Ptr("mountpoint"),
				Name:             utils.Ptr("testname"),
				PerformanceClass: &sfs.ResourcePoolPerformanceClass{
					Name:       utils.Ptr("performance"),
					PeakIops:   utils.Ptr[int64](42),
					Throughput: utils.Ptr[int64](54),
				},
				PerformanceClassDowngradableAt: utils.Ptr(now),
				SizeReducibleAt:                utils.Ptr(now.Add(1 * time.Hour)),
				Space: &sfs.ResourcePoolSpace{
					SizeGigabytes: utils.Ptr[int64](42),
				},
				State: utils.Ptr("state"),
			},
			expected: &dataSourceModel{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("foo"),
					types.StringValue("bar"),
					types.StringValue("baz"),
				}),
				Name:                           types.StringValue("testname"),
				PerformanceClass:               types.StringValue("performance"),
				SizeGigabytes:                  types.Int64Value(42),
				Region:                         testRegion,
				SizeReducibleAt:                testTimePlus1h,
				PerformanceClassDowngradableAt: testTime,
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapDataSourceFields(ctx, tt.region, tt.input, tt.state); (err == nil) != tt.isValid {
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
