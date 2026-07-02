package observability

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	observabilitySdk "github.com/stackitcloud/stackit-sdk-go/services/observability/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *observabilitySdk.Credentials
		expected    Model
		isValid     bool
	}{
		{
			"ok",
			&observabilitySdk.Credentials{
				Username: "username",
				Password: "password",
			},
			Model{
				Id:                types.StringValue("pid,iid,username"),
				ProjectId:         types.StringValue("pid"),
				InstanceId:        types.StringValue("iid"),
				Username:          types.StringValue("username"),
				Password:          types.StringValue("password"),
				RotateWhenChanged: types.MapNull(types.StringType),
			},
			true,
		},
		{
			"response_nil_fail",
			nil,
			Model{},
			false,
		},
		{
			"response_fields_nil_fail",
			&observabilitySdk.Credentials{
				Password: "",
				Username: "",
			},
			Model{},
			false,
		},
		{
			"no_resource_id",
			&observabilitySdk.Credentials{},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId:         tt.expected.ProjectId,
				InstanceId:        tt.expected.InstanceId,
				RotateWhenChanged: types.MapNull(types.StringType),
			}
			err := mapFields(tt.input, state)
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
