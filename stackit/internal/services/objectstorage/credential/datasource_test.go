package objectstorage

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/objectstorage"
)

func TestMapDatasourceFields(t *testing.T) {
	now := time.Now()

	tests := []struct {
		description string
		input       *objectstorage.AccessKey
		expected    DataSourceModel
		isValid     bool
	}{
		{
			"default_values",
			&objectstorage.AccessKey{},
			DataSourceModel{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				ExpirationTimestamp: types.StringNull(),
				Region:              types.StringValue("eu01"),
			},
			true,
		},
		{
			"simple_values",
			&objectstorage.AccessKey{
				DisplayName: utils.Ptr("name"),
				Expires:     utils.Ptr(now.Format(time.RFC3339)),
			},
			DataSourceModel{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue("name"),
				ExpirationTimestamp: types.StringValue(now.Format(time.RFC3339)),
				Region:              types.StringValue("eu01"),
			},
			true,
		},
		{
			"empty_strings",
			&objectstorage.AccessKey{
				DisplayName: utils.Ptr(""),
			},
			DataSourceModel{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringValue(""),
				ExpirationTimestamp: types.StringNull(),
				Region:              types.StringValue("eu01"),
			},
			true,
		},
		{
			"expiration_timestamp_with_fractional_seconds",
			&objectstorage.AccessKey{
				Expires: utils.Ptr(now.Format(time.RFC3339Nano)),
			},
			DataSourceModel{
				Id:                  types.StringValue("pid,cgid,cid"),
				ProjectId:           types.StringValue("pid"),
				CredentialsGroupId:  types.StringValue("cgid"),
				CredentialId:        types.StringValue("cid"),
				Name:                types.StringNull(),
				ExpirationTimestamp: types.StringValue(now.Format(time.RFC3339)),
				Region:              types.StringValue("eu01"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			DataSourceModel{},
			false,
		},
		{
			"bad_time",
			&objectstorage.AccessKey{
				Expires: utils.Ptr("foo-bar"),
			},
			DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &DataSourceModel{
				ProjectId:          tt.expected.ProjectId,
				CredentialsGroupId: tt.expected.CredentialsGroupId,
				CredentialId:       tt.expected.CredentialId,
			}
			err := mapDataSourceFields(tt.input, model, "eu01")
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
