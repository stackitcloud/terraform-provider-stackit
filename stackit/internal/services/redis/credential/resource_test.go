package redis

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	redis "github.com/stackitcloud/stackit-sdk-go/services/redis/v1api"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		state       Model
		input       *redis.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&redis.CredentialsResponse{
				Id:  "cid",
				Raw: &redis.RawCredentials{},
			},
			Model{
				Id:               types.StringValue("pid,iid,cid"),
				CredentialId:     types.StringValue("cid"),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Host:             types.StringValue(""),
				Hosts:            types.ListNull(types.StringType),
				LoadBalancedHost: types.StringNull(),
				Password:         types.StringValue(""),
				Port:             types.Int32Null(),
				Uri:              types.StringNull(),
				Username:         types.StringValue(""),
			},
			true,
		},
		{
			"simple_values",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&redis.CredentialsResponse{
				Id: "cid",
				Raw: &redis.RawCredentials{
					Credentials: redis.Credentials{
						Host: "host",
						Hosts: []string{
							"host_1",
							"",
						},
						LoadBalancedHost: new("load_balanced_host"),
						Password:         "password",
						Port:             new(int32(1234)),
						Uri:              new("uri"),
						Username:         "username",
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
				LoadBalancedHost: types.StringValue("load_balanced_host"),
				Password:         types.StringValue("password"),
				Port:             types.Int32Value(1234),
				Uri:              types.StringValue("uri"),
				Username:         types.StringValue("username"),
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
			&redis.CredentialsResponse{
				Id: "cid",
				Raw: &redis.RawCredentials{
					Credentials: redis.Credentials{
						Host: "host",
						Hosts: []string{
							"",
							"host_1",
							"host_2",
						},
						LoadBalancedHost: new("load_balanced_host"),
						Password:         "password",
						Port:             new(int32(1234)),
						Uri:              new("uri"),
						Username:         "username",
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
				LoadBalancedHost: types.StringValue("load_balanced_host"),
				Password:         types.StringValue("password"),
				Port:             types.Int32Value(1234),
				Uri:              types.StringValue("uri"),
				Username:         types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&redis.CredentialsResponse{
				Id: "cid",
				Raw: &redis.RawCredentials{
					Credentials: redis.Credentials{
						Host:             "",
						Hosts:            []string{},
						LoadBalancedHost: nil,
						Password:         "",
						Port:             new(int32(2123456789)),
						Uri:              nil,
						Username:         "",
					},
				},
			},
			Model{
				Id:               types.StringValue("pid,iid,cid"),
				CredentialId:     types.StringValue("cid"),
				InstanceId:       types.StringValue("iid"),
				ProjectId:        types.StringValue("pid"),
				Host:             types.StringValue(""),
				Hosts:            types.ListValueMust(types.StringType, []attr.Value{}),
				LoadBalancedHost: types.StringNull(),
				Password:         types.StringValue(""),
				Port:             types.Int32Value(2123456789),
				Uri:              types.StringNull(),
				Username:         types.StringValue(""),
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
			&redis.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			Model{
				InstanceId: types.StringValue("iid"),
				ProjectId:  types.StringValue("pid"),
			},
			&redis.CredentialsResponse{
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
