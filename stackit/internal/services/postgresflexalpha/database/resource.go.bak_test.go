package postgresflexa

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/postgresflex"
)

func TestMapFields(t *testing.T) {
	const testRegion = "region"
	tests := []struct {
		description string
		input       *postgresflex.InstanceDatabase
		region      string
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&postgresflex.InstanceDatabase{
				Id: utils.Ptr("uid"),
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				DatabaseId: types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringNull(),
				Owner:      types.StringNull(),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"simple_values",
			&postgresflex.InstanceDatabase{
				Id:   utils.Ptr("uid"),
				Name: utils.Ptr("dbname"),
				Options: &map[string]interface{}{
					"owner": "username",
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				DatabaseId: types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue("dbname"),
				Owner:      types.StringValue("username"),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&postgresflex.InstanceDatabase{
				Id:   utils.Ptr("uid"),
				Name: utils.Ptr(""),
				Options: &map[string]interface{}{
					"owner": "",
				},
			},
			testRegion,
			Model{
				Id:         types.StringValue("pid,region,iid,uid"),
				DatabaseId: types.StringValue("uid"),
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Name:       types.StringValue(""),
				Owner:      types.StringValue(""),
				Region:     types.StringValue(testRegion),
			},
			true,
		},
		{
			"nil_response",
			nil,
			testRegion,
			Model{},
			false,
		},
		{
			"empty_response",
			&postgresflex.InstanceDatabase{},
			testRegion,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&postgresflex.InstanceDatabase{
				Id:   utils.Ptr(""),
				Name: utils.Ptr("dbname"),
				Options: &map[string]interface{}{
					"owner": "username",
				},
			},
			testRegion,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, state, tt.region)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(state, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *postgresflex.CreateDatabasePayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{
				Name:  types.StringValue("dbname"),
				Owner: types.StringValue("username"),
			},
			&postgresflex.CreateDatabasePayload{
				Name: utils.Ptr("dbname"),
				Options: &map[string]string{
					"owner": "username",
				},
			},
			true,
		},
		{
			"null_fields",
			&Model{
				Name:  types.StringNull(),
				Owner: types.StringNull(),
			},
			&postgresflex.CreateDatabasePayload{
				Name: nil,
				Options: &map[string]string{
					"owner": "",
				},
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
