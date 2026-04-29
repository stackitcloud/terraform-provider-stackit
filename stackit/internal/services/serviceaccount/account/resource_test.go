package account

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/types"
	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *Model
		expected    *serviceaccount.CreateServiceAccountPayload
		isValid     bool
	}{
		{
			"default_values",
			&Model{
				Name: types.StringValue("example-name1"),
			},
			&serviceaccount.CreateServiceAccountPayload{
				Name: "example-name1",
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
			output, err := toCreatePayload(tt.input)
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
		input       *serviceaccount.ServiceAccount
		expected    Model
		isValid     bool
	}{
		{
			"default_values",
			&serviceaccount.ServiceAccount{
				Id:        "550e8400-e29b-41d4-a716-446655440000",
				ProjectId: "pid",
				Email:     "mail",
			},
			Model{
				Id:               types.StringValue("pid,mail"),
				ProjectId:        types.StringValue("pid"),
				ServiceAccountId: types.StringValue("550e8400-e29b-41d4-a716-446655440000"),
				Email:            types.StringValue("mail"),
			},
			true,
		},
		{
			"nil_response",
			nil,
			Model{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			state := &Model{
				ProjectId: tt.expected.ProjectId,
			}
			err := mapFields(tt.input, state)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(*state, tt.expected, cmp.AllowUnexported(types.String{}))
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
