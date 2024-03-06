package redis

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/redis"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *redis.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&redis.CredentialsResponse{
				Id:  utils.Ptr("cid"),
				Raw: &redis.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringNull(),
				Hosts:        types.ListNull(types.StringType),
				HttpAPIURI:   types.StringNull(),
				Name:         types.StringNull(),
				Password:     types.StringNull(),
				Port:         types.Int64Null(),
				Uri:          types.StringNull(),
				Username:     types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&redis.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &redis.RawCredentials{
					Credentials: &redis.Credentials{
						Host: utils.Ptr("host"),
						Hosts: &[]string{
							"host_1",
							"",
						},
						HttpApiUri: utils.Ptr("http"),
						Name:       utils.Ptr("name"),
						Password:   utils.Ptr("password"),
						Port:       utils.Ptr(int64(1234)),
						Uri:        utils.Ptr("uri"),
						Username:   utils.Ptr("username"),
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
				HttpAPIURI: types.StringValue("http"),
				Name:       types.StringValue("name"),
				Password:   types.StringValue("password"),
				Port:       types.Int64Value(1234),
				Uri:        types.StringValue("uri"),
				Username:   types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&redis.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &redis.RawCredentials{
					Credentials: &redis.Credentials{
						Host:       utils.Ptr(""),
						Hosts:      &[]string{},
						HttpApiUri: nil,
						Name:       nil,
						Password:   utils.Ptr(""),
						Port:       utils.Ptr(int64(2123456789)),
						Uri:        nil,
						Username:   utils.Ptr(""),
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
				HttpAPIURI:   types.StringNull(),
				Name:         types.StringNull(),
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
			&redis.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			&redis.CredentialsResponse{
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
