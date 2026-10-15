package machinetypes

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
)

func filterValue(t *testing.T, filter *filterModel) types.Object {
	t.Helper()
	if filter.ExtraSpecs.IsNull() {
		filter.ExtraSpecs = types.MapNull(types.StringType)
	}
	var resp datasource.SchemaResponse
	NewMachineTypesDataSource().Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	filterType, ok := resp.Schema.Attributes["filter"].GetType().(types.ObjectType)
	if !ok {
		t.Fatal("expected object filter type")
	}
	value, diags := types.ObjectValueFrom(context.Background(), filterType.AttrTypes, filter)
	if diags.HasError() {
		t.Fatal(diags)
	}
	return value
}

func TestBuildFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter filterModel
		want   string
	}{
		{
			name: "empty",
		},
		{
			name: "vcpu",
			filter: filterModel{
				VCPU: types.Int64Value(1),
			},
			want: "vcpus == 1",
		},
		{
			name: "ram",
			filter: filterModel{
				RAM: types.Int64Value(1024),
			},
			want: "ram == 1024",
		},
		{
			name: "vcpu and ram",
			filter: filterModel{
				VCPU: types.Int64Value(1),
				RAM:  types.Int64Value(1024),
			},
			want: "ram == 1024 && vcpus == 1",
		},
		{
			name: "zero disk",
			filter: filterModel{
				Disk: types.Int64Value(0),
			},
			want: "disk == 0",
		},
		{
			name: "empty name",
			filter: filterModel{
				Name: types.StringValue(""),
			},
			want: `name == ""`,
		},
		{
			name: "all filters",
			filter: filterModel{
				Name: types.StringValue("t1.2"),
				Disk: types.Int64Value(1),
				RAM:  types.Int64Value(1024),
				VCPU: types.Int64Value(1),
				ExtraSpecs: types.MapValueMust(types.StringType, map[string]attr.Value{
					"overcommit": types.StringValue("4"),
					"cpu":        types.StringValue("intel-icelake-generic"),
				}),
			},
			want: `name == "t1.2" && disk == 1 && ram == 1024 && vcpus == 1 && extraSpecs["cpu"] == "intel-icelake-generic" && extraSpecs["overcommit"] == "4"`,
		},
		{
			name: "escaped strings and map keys",
			filter: filterModel{
				Name: types.StringValue("quoted\"\\\n"),
				ExtraSpecs: types.MapValueMust(types.StringType, map[string]attr.Value{
					`cpu"] || true || extraSpecs["cpu`: types.StringValue(`intel" || true`),
				}),
			},
			want: `name == "quoted\"\\\n" && extraSpecs["cpu\"] || true || extraSpecs[\"cpu"] == "intel\" || true"`,
		},
		{
			name: "empty extra specs",
			filter: filterModel{
				ExtraSpecs: types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFilter(context.Background(), filterValue(t, &tt.filter))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("filter = %q, want %q", got, tt.want)
			}
		})
	}
	for _, value := range []types.Object{types.ObjectNull(nil), types.ObjectUnknown(nil)} {
		if got, err := buildFilter(context.Background(), value); err != nil || got != "" {
			t.Fatalf("absent filter = %q, %v", got, err)
		}
	}
}

func TestBuildFilterErrors(t *testing.T) {
	for _, value := range []types.String{types.StringNull(), types.StringUnknown()} {
		filter := filterValue(t, &filterModel{ExtraSpecs: types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": value})})
		if _, err := buildFilter(context.Background(), filter); err == nil {
			t.Fatal("expected error for an unknown or null extra spec")
		}
	}
	malformed := types.ObjectValueMust(map[string]attr.Type{"unexpected": types.StringType}, map[string]attr.Value{"unexpected": types.StringValue("value")})
	if _, err := buildFilter(context.Background(), malformed); err == nil {
		t.Fatal("expected error for malformed filter")
	}
}

func TestMapFields(t *testing.T) {
	ctx := context.Background()
	response := &iaas.MachineTypeListResponse{Items: []iaas.MachineType{
		{
			Name:  "t1.2",
			Disk:  0,
			Ram:   2048,
			Vcpus: 2,
		},
		{
			Name:        "t1.1",
			Description: new("small"),
			Disk:        1,
			Ram:         1024,
			Vcpus:       1,
			ExtraSpecs:  map[string]any{"cpu": "intel-icelake-generic"},
		},
	}}
	filter := filterValue(t, &filterModel{VCPU: types.Int64Value(1)})
	model := DataSourceModel{
		ProjectID: types.StringValue("project"),
		Filter:    filter,
	}
	if err := mapFields(ctx, response, &model, "eu01"); err != nil {
		t.Fatal(err)
	}
	wantResults, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: machineTypeAttrTypes}, []machineTypeModel{
		{
			Name:        types.StringValue("t1.1"),
			Description: types.StringValue("small"),
			Disk:        types.Int64Value(1),
			RAM:         types.Int64Value(1024),
			VCPUs:       types.Int64Value(1),
			ExtraSpecs:  types.MapValueMust(types.StringType, map[string]attr.Value{"cpu": types.StringValue("intel-icelake-generic")}),
		},
		{
			Name:        types.StringValue("t1.2"),
			Description: types.StringNull(),
			Disk:        types.Int64Value(0),
			RAM:         types.Int64Value(2048),
			VCPUs:       types.Int64Value(2),
			ExtraSpecs:  types.MapNull(types.StringType),
		},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := DataSourceModel{
		ID:        types.StringValue("project,eu01"),
		ProjectID: types.StringValue("project"),
		Region:    types.StringValue("eu01"),
		Filter:    filter,
		Results:   wantResults,
	}
	if diff := cmp.Diff(want, model); diff != "" {
		t.Fatalf("mapped data does not match (-want +got):\n%s", diff)
	}
	if response.Items[0].Name != "t1.2" {
		t.Fatal("mapping should not mutate the API response")
	}
}

func TestMapFieldsEmpty(t *testing.T) {
	for _, items := range [][]iaas.MachineType{nil, {}} {
		model := DataSourceModel{ProjectID: types.StringValue("project")}
		if err := mapFields(context.Background(), &iaas.MachineTypeListResponse{Items: items}, &model, "eu01"); err != nil {
			t.Fatal(err)
		}
		if model.Results.IsNull() || model.Results.IsUnknown() || len(model.Results.Elements()) != 0 || model.ID.ValueString() != "project,eu01" {
			t.Fatalf("expected empty results with stable ID, got %v", model)
		}
	}
}

func TestMapFieldsErrors(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response *iaas.MachineTypeListResponse
		model    *DataSourceModel
	}{
		{
			name:  "nil response",
			model: &DataSourceModel{},
		},
		{
			name:     "nil model",
			response: &iaas.MachineTypeListResponse{},
		},
		{
			name: "missing name",
			response: &iaas.MachineTypeListResponse{
				Items: []iaas.MachineType{{}},
			},
			model: &DataSourceModel{},
		},
		{
			name: "invalid extra specs",
			response: &iaas.MachineTypeListResponse{
				Items: []iaas.MachineType{
					{
						Name:       "invalid",
						ExtraSpecs: map[string]any{"cpu": true},
					},
				},
			},
			model: &DataSourceModel{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := mapFields(context.Background(), tt.response, tt.model, "eu01"); err == nil {
				t.Fatal("expected mapping error")
			}
		})
	}
}
