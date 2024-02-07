package loadbalancer

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	"github.com/stackitcloud/stackit-sdk-go/services/loadbalancer"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *loadbalancer.CreateCredentialsPayload
		isValid     bool
	}{
		{
			"default_values_ok",
			&Model{},
			&loadbalancer.CreateCredentialsPayload{
				DisplayName: nil,
				Username:    nil,
				Password:    nil,
			},
			true,
		},
		{
			"simple_values_ok",
			&Model{
				DisplayName: types.StringValue("display_name"),
				Username:    types.StringValue("username"),
				Password:    types.StringValue("password"),
			},
			&loadbalancer.CreateCredentialsPayload{
				DisplayName: utils.Ptr("display_name"),
				Username:    utils.Ptr("username"),
				Password:    utils.Ptr("password"),
			},
			true,
		},
		{
			"nil_model",
			nil,
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output, err := toCreatePayload(context.Background(), tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}

func TestMapFields(t *testing.T) {
	tests := []struct {
		description string
		input       *loadbalancer.CredentialsResponse
		expected    *Model
		isValid     bool
	}{
		{
			"default_values_ok",
			&loadbalancer.CredentialsResponse{
				CredentialsRef: utils.Ptr("credentials_ref"),
				Username:       utils.Ptr("username"),
			},
			&Model{
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
			&Model{
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
			&Model{},
			false,
		},
		{
			"no_username",
			&loadbalancer.CredentialsResponse{
				CredentialsRef: utils.Ptr("credentials_ref"),
				DisplayName:    utils.Ptr("display_name"),
			},
			&Model{},
			false,
		},
		{
			"no_credentials_ref",
			&loadbalancer.CredentialsResponse{
				DisplayName: utils.Ptr("display_name"),
				Username:    utils.Ptr("username"),
			},
			&Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			model := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(context.Background(), tt.input, model)
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
