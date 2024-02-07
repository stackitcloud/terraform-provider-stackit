package loadbalancer

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
)

func TestMapDataSourceFields(t *testing.T) {
	tests := []struct {
		description string
		input       *loadbalancer.CredentialsResponse
		expected    *DataSourceModel
		isValid     bool
	}{
		{
			"default_values_ok",
			&loadbalancer.CredentialsResponse{
				CredentialsRef: utils.Ptr("credentials_ref"),
				Username:       utils.Ptr("username"),
			},
			&DataSourceModel{
				Id:             types.StringValue("pid,credentials_ref"),
				ProjectId:      types.StringValue("pid"),
				CredentialsRef: types.StringValue("credentials_ref"),
				Username:       types.StringValue("username"),
			},
			true,
		},

		{
			"simple_values_ok",
			&loadbalancer.CredentialsResponse{
				CredentialsRef: utils.Ptr("credentials_ref"),
				DisplayName:    utils.Ptr("display_name"),
				Username:       utils.Ptr("username"),
			},
			&DataSourceModel{
				Id:             types.StringValue("pid,credentials_ref"),
				ProjectId:      types.StringValue("pid"),
				CredentialsRef: types.StringValue("credentials_ref"),
				DisplayName:    types.StringValue("display_name"),
				Username:       types.StringValue("username"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			&DataSourceModel{},
			false,
		},
		{
			"no_username",
			&loadbalancer.CredentialsResponse{
				CredentialsRef: utils.Ptr("credentials_ref"),
				DisplayName:    utils.Ptr("display_name"),
			},
			&DataSourceModel{},
			false,
		},
		{
			"no_credentials_ref",
			&loadbalancer.CredentialsResponse{
				DisplayName: utils.Ptr("display_name"),
				Username:    utils.Ptr("username"),
			},
			&DataSourceModel{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &DataSourceModel{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapDataSourceFields(context.Background(), tt.input, model)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(model, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
