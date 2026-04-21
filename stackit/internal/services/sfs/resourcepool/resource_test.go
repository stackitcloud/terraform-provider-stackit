package resourcepool

import (
	"context"
	_ "embed"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
)

var (
	testProjectId        = types.StringValue(uuid.NewString())
	testResourcePoolId   = types.StringValue(uuid.NewString())
	testRegion           = types.StringValue("eu01")
	testId               = types.StringValue(testProjectId.ValueString() + "," + testRegion.ValueString() + "," + testResourcePoolId.ValueString())
	testAvailabilityZone = types.StringValue("some zone")
	testIpAcl            = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("foo"), types.StringValue("bar"), types.StringValue("baz")})
)

func TestMapFields(t *testing.T) {
	testTime := time.Now()
	tests := []struct {
		name     string
		state    *Model
		region   string
		input    *sfs.ResourcePool
		expected *Model
		isValid  bool
	}{
		{
			"default_values",
			&Model{
				Id:        testId,
				ProjectId: testProjectId,
			},
			testRegion.ValueString(),
			&sfs.ResourcePool{
				Id: testResourcePoolId.ValueStringPointer(),
			},
			&Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: types.StringNull(),
				IpAcl:            types.ListNull(types.StringType),
				Name:             types.StringNull(),
				PerformanceClass: types.StringNull(),
				SizeGigabytes:    types.Int32Null(),
				Region:           testRegion,
			},
			true,
		},
		{
			name: "simple_values",
			state: &Model{
				Id:        testId,
				ProjectId: testProjectId,
			},
			region: testRegion.ValueString(),
			input: &sfs.ResourcePool{
				AvailabilityZone: testAvailabilityZone.ValueStringPointer(),
				CountShares:      utils.Ptr[int32](42),
				CreatedAt:        &testTime,
				Id:               testResourcePoolId.ValueStringPointer(),
				IpAcl:            []string{"foo", "bar", "baz"},
				MountPath:        new("mountpoint"),
				Name:             new("testname"),
				PerformanceClass: &sfs.ResourcePoolPerformanceClass{
					Name:       new("performance"),
					PeakIops:   utils.Ptr[int32](42),
					Throughput: utils.Ptr[int32](54),
				},
				PerformanceClassDowngradableAt: &testTime,
				SizeReducibleAt:                &testTime,
				Space: &sfs.ResourcePoolSpace{
					SizeGigabytes: utils.Ptr[int32](42),
				},
				State: new("state"),
			},
			expected: &Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("foo"),
					types.StringValue("bar"),
					types.StringValue("baz"),
				}),
				Name:             types.StringValue("testname"),
				PerformanceClass: types.StringValue("performance"),
				SizeGigabytes:    types.Int32Value(42),
				Region:           testRegion,
			},
			isValid: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := mapFields(ctx, tt.region, tt.input, tt.state); (err == nil) != tt.isValid {
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
		want    *sfs.CreateResourcePoolPayload
		wantErr bool
	}{
		{
			"default",
			&Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl:            testIpAcl,
				Name:             types.StringValue("testname"),
				PerformanceClass: types.StringValue("performance"),
				SizeGigabytes:    types.Int32Value(42),
			},
			&sfs.CreateResourcePoolPayload{
				AvailabilityZone: testAvailabilityZone.ValueString(),
				IpAcl:            []string{"foo", "bar", "baz"},
				Name:             "testname",
				PerformanceClass: "performance",
				SizeGigabytes:    42,
			},
			false,
		},
		{
			"undefined ACL",
			&Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl:            types.ListNull(types.StringType),
				Name:             types.StringValue("testname"),
				PerformanceClass: types.StringValue("performance"),
				SizeGigabytes:    types.Int32Value(42),
			},
			&sfs.CreateResourcePoolPayload{
				AvailabilityZone: testAvailabilityZone.ValueString(),
				IpAcl:            nil,
				Name:             "testname",
				PerformanceClass: "performance",
				SizeGigabytes:    42,
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toCreatePayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUpdatePayload(t *testing.T) {
	tests := []struct {
		name    string
		model   *Model
		want    *sfs.UpdateResourcePoolPayload
		wantErr bool
	}{
		{
			"default",
			&Model{
				Id:                  testId,
				ProjectId:           testProjectId,
				ResourcePoolId:      testResourcePoolId,
				AvailabilityZone:    testAvailabilityZone,
				IpAcl:               testIpAcl,
				Name:                types.StringValue("testname"),
				PerformanceClass:    types.StringValue("performance"),
				SizeGigabytes:       types.Int32Value(42),
				SnapshotsAreVisible: types.BoolValue(true),
			},
			&sfs.UpdateResourcePoolPayload{
				IpAcl:               []string{"foo", "bar", "baz"},
				PerformanceClass:    new("performance"),
				SizeGigabytes:       *sfs.NewNullableInt32(utils.Ptr[int32](42)),
				SnapshotsAreVisible: new(true),
			},
			false,
		},
		{
			"undefined ACL",
			&Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl:            types.ListNull(types.StringType),
				Name:             types.StringValue("testname"),
				PerformanceClass: types.StringValue("performance"),
				SizeGigabytes:    types.Int32Value(42),
			},
			&sfs.UpdateResourcePoolPayload{
				IpAcl:            nil,
				PerformanceClass: new("performance"),
				SizeGigabytes:    *sfs.NewNullableInt32(utils.Ptr[int32](42)),
			},
			false,
		},
		{
			"empty ACL",
			&Model{
				Id:               testId,
				ProjectId:        testProjectId,
				ResourcePoolId:   testResourcePoolId,
				AvailabilityZone: testAvailabilityZone,
				IpAcl:            types.ListValueMust(types.StringType, []attr.Value{}),
				Name:             types.StringValue("testname"),
				PerformanceClass: types.StringValue("performance"),
				SizeGigabytes:    types.Int32Value(42),
			},
			&sfs.UpdateResourcePoolPayload{
				IpAcl:            []string{},
				PerformanceClass: new("performance"),
				SizeGigabytes:    *sfs.NewNullableInt32(utils.Ptr[int32](42)),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toUpdatePayload(tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("toUpdatePayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toUpdatePayload() = %v, want %v", got, tt.want)
			}
		})
	}
}
