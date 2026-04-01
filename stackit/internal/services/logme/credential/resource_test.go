package logme

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/logme"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logme.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&logme.CredentialsResponse{
				Id:  new("cid"),
				Raw: &logme.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringNull(),
				Password:     types.StringNull(),
				Port:         types.Int64Null(),
				Uri:          types.StringNull(),
				Username:     types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&logme.CredentialsResponse{
				Id: new("cid"),
				Raw: &logme.RawCredentials{
					Credentials: &logme.Credentials{
						Host:     new("host"),
						Password: new("password"),
						Port:     new(int64(1234)),
						Uri:      new("uri"),
						Username: new("username"),
					},
				},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue("host"),
				Password:     types.StringValue("password"),
				Port:         types.Int64Value(1234),
				Uri:          types.StringValue("uri"),
				Username:     types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&logme.CredentialsResponse{
				Id: new("cid"),
				Raw: &logme.RawCredentials{
					Credentials: &logme.Credentials{
						Host:     new(""),
						Password: new(""),
						Port:     new(int64(2123456789)),
						Uri:      nil,
						Username: new(""),
					},
				},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue(""),
				Password:     types.StringValue(""),
				Port:         types.Int64Value(2123456789),
				Uri:          types.StringNull(),
				Username:     types.StringValue(""),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			&logme.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			&logme.CredentialsResponse{
				Id: new("cid"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId:  tt.expected.ProjectId,
				InstanceId: tt.expected.InstanceId,
			}
			err := mapFields(tt.input, model)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, &tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
