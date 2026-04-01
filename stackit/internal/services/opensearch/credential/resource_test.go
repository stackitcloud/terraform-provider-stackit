package opensearch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/opensearch"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *opensearch.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id:  new("cid"),
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
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id: new("cid"),
				Raw: &opensearch.RawCredentials{
					Credentials: &opensearch.Credentials{
						Host: new("host"),
						Hosts: &[]string{
							"host_1",
							"",
						},
						Password: new("password"),
						Port:     new(int64(1234)),
						Scheme:   new("scheme"),
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
			"hosts_unordered",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
				Hosts: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("host_2"),
					types.StringValue(""),
					types.StringValue("host_1"),
				}),
			},
			&opensearch.CredentialsResponse{
				Id: new("cid"),
				Raw: &opensearch.RawCredentials{
					Credentials: &opensearch.Credentials{
						Host: new("host"),
						Hosts: &[]string{
							"",
							"host_1",
							"host_2",
						},
						Password: new("password"),
						Port:     new(int64(1234)),
						Scheme:   new("scheme"),
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
				Hosts: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("host_2"),
					types.StringValue(""),
					types.StringValue("host_1"),
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
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id: new("cid"),
				Raw: &opensearch.RawCredentials{
					Credentials: &opensearch.Credentials{
						Host:     new(""),
						Hosts:    &[]string{},
						Password: new(""),
						Port:     new(int64(2123456789)),
						Scheme:   nil,
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
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			nil,
			Model{},
			false,
		},
		{
			"no_resource_id",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id: new("cid"),
			},
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapFields(context.Background(), tt.input, &tt.state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(tt.state, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
