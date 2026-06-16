package opensearch

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	opensearch "github.com/stackitcloud/stackit-sdk-go/services/opensearch/v1api"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		state       DataSourceModel
		input       *opensearch.CredentialsResponse
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			DataSourceModel{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id:  "cid",
				Raw: &opensearch.RawCredentials{},
			},
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
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
			DataSourceModel{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			nil,
			DataSourceModel{},
			false,
		},
		{
			"no_resource_id",
			DataSourceModel{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{},
			DataSourceModel{},
			false,
		},
		{
			"nil_raw_credential",
			DataSourceModel{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&opensearch.CredentialsResponse{
				Id: "cid",
			},
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			err := mapDataSourceFields(context.Background(), tt.input, &tt.state)
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
