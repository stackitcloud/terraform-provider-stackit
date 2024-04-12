package opensearch

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *opensearch.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&opensearch.CredentialsResponse{
				Id:  utils.Ptr("cid"),
				Raw: &opensearch.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringNull(),
				Hosts:        types.ListNull(types.StringType),
				Password:     types.StringNull(),
				Port:         types.Int64Null(),
				Scheme:       types.StringNull(),
				Uri:          types.StringNull(),
				Username:     types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&opensearch.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &opensearch.RawCredentials{
					Credentials: &opensearch.Credentials{
						Host: utils.Ptr("host"),
						Hosts: &[]string{
							"host_1",
							"",
						},
						Password: utils.Ptr("password"),
						Port:     utils.Ptr(int64(1234)),
						Scheme:   utils.Ptr("scheme"),
						Uri:      utils.Ptr("uri"),
						Username: utils.Ptr("username"),
					},
				},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue("host"),
				Hosts: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("host_1"),
					types.StringValue(""),
				}),
				Password: types.StringValue("password"),
				Port:     types.Int64Value(1234),
				Scheme:   types.StringValue("scheme"),
				Uri:      types.StringValue("uri"),
				Username: types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&opensearch.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &opensearch.RawCredentials{
					Credentials: &opensearch.Credentials{
						Host:     utils.Ptr(""),
						Hosts:    &[]string{},
						Password: utils.Ptr(""),
						Port:     utils.Ptr(int64(2123456789)),
						Scheme:   nil,
						Uri:      nil,
						Username: utils.Ptr(""),
					},
				},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue(""),
				Hosts:        types.ListValueMust(types.StringType, []attr.Value{}),
				Password:     types.StringValue(""),
				Port:         types.Int64Value(2123456789),
				Scheme:       types.StringNull(),
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
			&opensearch.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			&opensearch.CredentialsResponse{
				Id: utils.Ptr("cid"),
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
