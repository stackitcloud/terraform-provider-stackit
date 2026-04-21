package share

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	sfs "github.com/stackitcloud/stackit-sdk-go/services/sfs/v1api"
)

func TestMapDatasourceFields(t *testing.T) {
	tests := []struct {
		name     string
		state    *dataSourceModel
		region   string
		input    *sfs.Share
		expected *dataSourceModel
		isValid  bool
	}{
		{
			"default_values",
			&dataSourceModel{
				Id:             testId,
				ProjectId:      testProjectId,
				ResourcePoolId: testResourcePoolId,
			},
			"eu01",
			&sfs.Share{
				ExportPolicy: *sfs.NewNullableShareExportPolicy(&sfs.ShareExportPolicy{
					Id:   testId.ValueStringPointer(),
					Name: new("test-policy"),
				}),
				Id:                      testShareId.ValueStringPointer(),
				MountPath:               new("/testmount"),
				Name:                    new("test-name"),
				SpaceHardLimitGigabytes: utils.Ptr[int32](42),
			},
			&dataSourceModel{
				Id:                      testId,
				ProjectId:               testProjectId,
				ResourcePoolId:          testResourcePoolId,
				ShareId:                 testShareId,
				Name:                    types.StringValue("test-name"),
				ExportPolicyName:        testPolicyName,
				SpaceHardLimitGigabytes: types.Int32Value(42),
				MountPath:               types.StringValue("/testmount"),
				Region:                  testRegion,
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
