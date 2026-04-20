package secretsmanager

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	secretsmanager "github.com/stackitcloud/stackit-sdk-go/services/secretsmanager/v1api"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *secretsmanager.User
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&secretsmanager.User{
				Id: "uid",
			},
			DataSourceModel{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringValue(""),
				WriteEnabled: types.BoolValue(false),
				Username:     types.StringValue(""),
			},
			true,
		},
		{
			"simple_values",
			&secretsmanager.User{
				Id:          "uid",
				Description: "description",
				Write:       false,
				Username:    "username",
			},
			DataSourceModel{
				Id:           types.StringValue("pid,iid,uid"),
				UserId:       types.StringValue("uid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Description:  types.StringValue("description"),
				WriteEnabled: types.BoolValue(false),
				Username:     types.StringValue("username"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			&secretsmanager.User{},
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &DataSourceModel{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapDataSourceFields(tt.input, state)
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
