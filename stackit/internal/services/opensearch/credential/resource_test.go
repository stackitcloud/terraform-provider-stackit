package opensearch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	opensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch/v1api"
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
				Id:  "cid",
				Raw: &opensearch.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringValue(""),
				Hosts:        types.ListNull(types.StringType),
				Password:     types.StringValue(""),
				Port:         types.Int32Null(),
				Scheme:       types.StringNull(),
				Uri:          types.StringNull(),
				Username:     types.StringValue(""),
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
				Id: "cid",
				Raw: &opensearch.RawCredentials{
					Credentials: opensearch.Credentials{
						Host: "host",
						Hosts: []string{
							"host_1",
							"",
						},
						Password: "password",
						Port:     new(int32(1234)),
						Scheme:   new("scheme"),
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
				Hosts: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("host_1"),
					types.StringValue(""),
				}),
				Password: types.StringValue("password"),
				Port:     types.Int32Value(1234),
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
				Id: "cid",
				Raw: &opensearch.RawCredentials{
					Credentials: opensearch.Credentials{
						Host: "host",
						Hosts: []string{
							"",
							"host_1",
							"host_2",
						},
						Password: "password",
						Port:     new(int32(1234)),
						Scheme:   new("scheme"),
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
				Hosts: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("host_2"),
					types.StringValue(""),
					types.StringValue("host_1"),
				}),
				Password: types.StringValue("password"),
				Port:     types.Int32Value(1234),
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
				Id: "cid",
				Raw: &opensearch.RawCredentials{
					Credentials: opensearch.Credentials{
						Host:     "",
						Hosts:    []string{},
						Password: "",
						Port:     new(int32(2123456789)),
						Scheme:   nil,
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
				Hosts:        types.ListValueMust(types.StringType, []attr.Value{}),
				Password:     types.StringValue(""),
				Port:         types.Int32Value(2123456789),
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
				Id: "cid",
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
