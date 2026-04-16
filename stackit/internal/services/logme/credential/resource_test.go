package logme

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	logmeSdk "github.com/stackitcloud/stackit-sdk-go/services/logme/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *logmeSdk.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&logmeSdk.CredentialsResponse{
				Id:  "cid",
				Raw: &logmeSdk.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue(""),
				Password:     types.StringValue(""),
				Port:         types.Int32Null(),
				Uri:          types.StringNull(),
				Username:     types.StringValue(""),
			},
			true,
		},
		{
			"simple_values",
			&logmeSdk.CredentialsResponse{
				Id: "cid",
				Raw: &logmeSdk.RawCredentials{
					Credentials: logmeSdk.Credentials{
						Host:     "host",
						Password: "password",
						Port:     new(int32(1234)),
						Uri:      new("uri"),
						Username: "username",
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
				Port:         types.Int32Value(1234),
				Uri:          types.StringValue("uri"),
				Username:     types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&logmeSdk.CredentialsResponse{
				Id: "cid",
				Raw: &logmeSdk.RawCredentials{
					Credentials: logmeSdk.Credentials{
						Host:     "",
						Password: "",
						Port:     new(int32(2123456789)),
						Uri:      nil,
						Username: "",
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
				Port:         types.Int32Value(2123456789),
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
			&logmeSdk.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			&logmeSdk.CredentialsResponse{
				Id: "cid",
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
