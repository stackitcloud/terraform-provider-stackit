package snapshots

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
)

var (
	testProjectId      = types.StringValue(uuid.NewString())
	testResourcePoolId = types.StringValue(uuid.NewString())
	testRegion         = types.StringValue("eu01")
	testId             = types.StringValue(testProjectId.ValueString() + "," + testRegion.ValueString() + "," + testResourcePoolId.ValueString())
)

func must[T any](t T, diags diag.Diagnostics) T {
	if diags.HasError() {
		panic(fmt.Sprintf("diagnostics contain error: %v", diags.Errors()))
	}
	return t
}

func TestMapDatasourceFields(t *testing.T) {
	testTime := time.Now()
	tests := []struct {
		name     string
		state    *dataSourceModel
		region   string
		input    *[]sfs.ResourcePoolSnapshot
		expected *dataSourceModel
		isValid  bool
	}{
		{
			"default_values",
			&dataSourceModel{
				Id:             testId,
				ProjectId:      testProjectId,
				ResourcePoolId: testResourcePoolId,
				Region:         types.StringValue("eu01"),
			},
			"eu01",
			&[]sfs.ResourcePoolSnapshot{
				{
					Comment:              sfs.NewNullableString(utils.Ptr("comment 1")),
					CreatedAt:            utils.Ptr(testTime),
					ResourcePoolId:       testResourcePoolId.ValueStringPointer(),
					SnapshotName:         utils.Ptr("snapshot-1"),
					SizeGigabytes:        utils.Ptr(int64(50)),
					LogicalSizeGigabytes: utils.Ptr(int64(50)),
				},
				{
					Comment:              sfs.NewNullableString(utils.Ptr("comment 2")),
					CreatedAt:            utils.Ptr(testTime.Add(1 * time.Hour)),
					ResourcePoolId:       testResourcePoolId.ValueStringPointer(),
					SnapshotName:         utils.Ptr("snapshot-2"),
					SizeGigabytes:        utils.Ptr(int64(50)),
					LogicalSizeGigabytes: utils.Ptr(int64(50)),
				},
			},
			&dataSourceModel{
				Id:             testId,
				ProjectId:      testProjectId,
				ResourcePoolId: testResourcePoolId,
				Region:         types.StringValue("eu01"),
				Snapshots: types.ListValueMust(snapshotModelType, []attr.Value{
					must(types.ObjectValueFrom(context.Background(), snapshotModelType.AttrTypes, snapshotModel{
						Comment:              types.StringValue("comment 1"),
						CreatedAt:            types.StringValue(testTime.Format(time.RFC3339)),
						ResourcePoolId:       testResourcePoolId,
						SnapshotName:         types.StringValue("snapshot-1"),
						SizeGigabytes:        types.Int64Value(50),
						LogicalSizeGigabytes: types.Int64Value(50),
					})),
					must(types.ObjectValueFrom(context.Background(), snapshotModelType.AttrTypes, snapshotModel{
						Comment:              types.StringValue("comment 2"),
						CreatedAt:            types.StringValue(testTime.Add(1 * time.Hour).Format(time.RFC3339)),
						ResourcePoolId:       testResourcePoolId,
						SnapshotName:         types.StringValue("snapshot-2"),
						SizeGigabytes:        types.Int64Value(50),
						LogicalSizeGigabytes: types.Int64Value(50),
					})),
				}),
			},
			true,
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
