package rabbitmq

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/rabbitmq"
)

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *rabbitmq.CredentialsResponse
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&rabbitmq.CredentialsResponse{
				Id:  utils.Ptr("cid"),
				Raw: &rabbitmq.RawCredentials{},
			},
			Model{
				Id:           types.StringValue("pid,iid,cid"),
				CredentialId: types.StringValue("cid"),
				InstanceId:   types.StringValue("iid"),
				ProjectId:    types.StringValue("pid"),
				Host:         types.StringNull(),
				Hosts:        types.ListNull(types.StringType),
				HttpAPIURI:   types.StringNull(),
				HttpAPIURIs:  types.ListNull(types.StringType),
				Management:   types.StringNull(),
				Password:     types.StringNull(),
				Port:         types.Int64Null(),
				Uri:          types.StringNull(),
				Uris:         types.ListNull(types.StringType),
				Username:     types.StringNull(),
			},
			true,
		},
		{
			"simple_values",
			&rabbitmq.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &rabbitmq.RawCredentials{
					Credentials: &rabbitmq.Credentials{
						Host: utils.Ptr("host"),
						Hosts: &[]string{
							"host_1",
							"",
						},
						HttpApiUri: utils.Ptr("http"),
						HttpApiUris: &[]string{
							"http_api_uri_1",
							"",
						},
						Management: utils.Ptr("management"),
						Password:   utils.Ptr("password"),
						Port:       utils.Ptr(int64(1234)),
						Uri:        utils.Ptr("uri"),
						Uris: &[]string{
							"uri_1",
							"",
						},
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
				HttpAPIURI: types.StringValue("http"),
				HttpAPIURIs: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("http_api_uri_1"),
					types.StringValue(""),
				}),
				Management: types.StringValue("management"),
				Password:   types.StringValue("password"),
				Port:       types.Int64Value(1234),
				Uri:        types.StringValue("uri"),
				Uris: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("uri_1"),
					types.StringValue(""),
				}),
				Username: types.StringValue("username"),
			},
			true,
		},
		{
			"null_fields_and_int_conversions",
			&rabbitmq.CredentialsResponse{
				Id: utils.Ptr("cid"),
				Raw: &rabbitmq.RawCredentials{
					Credentials: &rabbitmq.Credentials{
						Host:        utils.Ptr(""),
						Hosts:       &[]string{},
						HttpApiUri:  nil,
						HttpApiUris: &[]string{},
						Management:  nil,
						Password:    utils.Ptr(""),
						Port:        utils.Ptr(int64(2123456789)),
						Uri:         nil,
						Uris:        &[]string{},
						Username:    utils.Ptr(""),
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
				HttpAPIURIs:  types.ListValueMust(types.StringType, []attr.Value{}),
				Management:   types.StringNull(),
				Password:     types.StringValue(""),
				Port:         types.Int64Value(2123456789),
				Uri:          types.StringNull(),
				Uris:         types.ListValueMust(types.StringType, []attr.Value{}),
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
			&rabbitmq.CredentialsResponse{},
			Model{},
			false,
		},
		{
			"nil_raw_credential",
			&rabbitmq.CredentialsResponse{
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
