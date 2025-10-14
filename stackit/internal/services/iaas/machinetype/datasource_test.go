package machineType

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		name        string
		initial     DataSourceModel
		input       *iaas.MachineType
		expected    DataSourceModel
		expectError bool
	}{
		{
			name: "valid simple values",
			initial: DataSourceModel{
				ProjectId: types.StringValue("pid"),
			},
			input: &iaas.MachineType{
				Name:        utils.Ptr("s1.2"),
				Description: utils.Ptr("general-purpose small"),
				Disk:        utils.Ptr(int64(20)),
				Ram:         utils.Ptr(int64(2048)),
				Vcpus:       utils.Ptr(int64(2)),
				ExtraSpecs: &map[string]interface{}{
					"cpu":         "amd-epycrome-7702",
					"overcommit":  "1",
					"environment": "general",
				},
			},
			expected: DataSourceModel{
				Id:          types.StringValue("pid,s1.2"),
				ProjectId:   types.StringValue("pid"),
				Name:        types.StringValue("s1.2"),
				Description: types.StringValue("general-purpose small"),
				Disk:        types.Int64Value(20),
				Ram:         types.Int64Value(2048),
				Vcpus:       types.Int64Value(2),
				ExtraSpecs: types.MapValueMust(types.StringType, map[string]attr.Value{
					"cpu":         types.StringValue("amd-epycrome-7702"),
					"overcommit":  types.StringValue("1"),
					"environment": types.StringValue("general"),
				}),
			},
			expectError: false,
		},
		{
			name: "missing name should fail",
			initial: DataSourceModel{
				ProjectId: types.StringValue("pid-456"),
			},
			input: &iaas.MachineType{
				Description: utils.Ptr("gp-medium"),
			},
			expected:    DataSourceModel{},
			expectError: true,
		},
		{
			name:        "nil machineType should fail",
			initial:     DataSourceModel{},
			input:       nil,
			expected:    DataSourceModel{},
			expectError: true,
		},
		{
			name: "empty extraSpecs should return null map",
			initial: DataSourceModel{
				ProjectId: types.StringValue("pid-789"),
			},
			input: &iaas.MachineType{
				Name:        utils.Ptr("m1.noextras"),
				Description: utils.Ptr("no extras"),
				Disk:        utils.Ptr(int64(10)),
				Ram:         utils.Ptr(int64(1024)),
				Vcpus:       utils.Ptr(int64(1)),
				ExtraSpecs:  &map[string]interface{}{},
			},
			expected: DataSourceModel{
				Id:          types.StringValue("pid-789,m1.noextras"),
				ProjectId:   types.StringValue("pid-789"),
				Name:        types.StringValue("m1.noextras"),
				Description: types.StringValue("no extras"),
				Disk:        types.Int64Value(10),
				Ram:         types.Int64Value(1024),
				Vcpus:       types.Int64Value(1),
				ExtraSpecs:  types.MapNull(types.StringType),
			},
			expectError: false,
		},
		{
			name: "nil extrasSpecs should return null map",
			initial: DataSourceModel{
				ProjectId: types.StringValue("pid-987"),
			},
			input: &iaas.MachineType{
				Name:        utils.Ptr("g1.nil"),
				Description: utils.Ptr("missing extras"),
				Disk:        utils.Ptr(int64(40)),
				Ram:         utils.Ptr(int64(8096)),
				Vcpus:       utils.Ptr(int64(4)),
				ExtraSpecs:  nil,
			},
			expected: DataSourceModel{
				Id:          types.StringValue("pid-987,g1.nil"),
				ProjectId:   types.StringValue("pid-987"),
				Name:        types.StringValue("g1.nil"),
				Description: types.StringValue("missing extras"),
				Disk:        types.Int64Value(40),
				Ram:         types.Int64Value(8096),
				Vcpus:       types.Int64Value(4),
				ExtraSpecs:  types.MapNull(types.StringType),
			},
			expectError: false,
		},
		{
			name: "invalid extraSpecs with non-string values",
			initial: DataSourceModel{
				ProjectId: types.StringValue("test-err"),
			},
			input: &iaas.MachineType{
				Name:        utils.Ptr("invalid"),
				Description: utils.Ptr("bad map"),
				Disk:        utils.Ptr(int64(10)),
				Ram:         utils.Ptr(int64(4096)),
				Vcpus:       utils.Ptr(int64(2)),
				ExtraSpecs: &map[string]interface{}{
					"cpu":   "intel",
					"burst": true, // not a string
					"gen":   8,    // not a string
				},
			},
			expected:    DataSourceModel{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, &tt.initial)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			diff := cmp.Diff(tt.expected, tt.initial)
			if diff != "" {
				t.Errorf("unexpected diff (-want +got):\n%s", diff)
			}

			// Extra sanity check for proper ID format
			if id := tt.initial.Id.ValueString(); !strings.HasPrefix(id, tt.initial.ProjectId.ValueString()+",") {
				t.Errorf("unexpected ID format: got %q", id)
			}
		})
	}
}

func TestSortMachineTypeByName(t *testing.T) {
	tests := []struct {
		name        string
		input       []*iaas.MachineType
		ascending   bool
		expected    []string
		expectError bool
	}{
		{
			name:      "ascending order",
			input:     []*iaas.MachineType{{Name: utils.Ptr("zeta")}, {Name: utils.Ptr("alpha")}, {Name: utils.Ptr("gamma")}},
			ascending: true,
			expected:  []string{"alpha", "gamma", "zeta"},
		},
		{
			name:      "descending order",
			input:     []*iaas.MachineType{{Name: utils.Ptr("zeta")}, {Name: utils.Ptr("alpha")}, {Name: utils.Ptr("gamma")}},
			ascending: false,
			expected:  []string{"zeta", "gamma", "alpha"},
		},
		{
			name:      "handles nil names",
			input:     []*iaas.MachineType{{Name: utils.Ptr("beta")}, nil, {Name: nil}, {Name: utils.Ptr("alpha")}},
			ascending: true,
			expected:  []string{"alpha", "beta"},
		},
		{
			name:        "empty input",
			input:       []*iaas.MachineType{},
			ascending:   true,
			expected:    nil,
			expectError: false,
		},
		{
			name:        "nil input",
			input:       nil,
			ascending:   true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted, err := sortMachineTypeByName(tt.input, tt.ascending)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result []string
			for _, mt := range sorted {
				if mt.Name != nil {
					result = append(result, *mt.Name)
				}
			}

			if diff := cmp.Diff(tt.expected, result); diff != "" {
				t.Errorf("unexpected sorted order (-want +got):\n%s", diff)
			}
		})
	}
}
