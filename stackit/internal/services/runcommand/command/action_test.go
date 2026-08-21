package runcommand

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stackitcloud/stackit-sdk-go/services/runcommand/v1api"
)

func TestToCreatePayload(t *testing.T) {
	tests := []struct {
		description string
		input       *runCommandModel
		expected    *v1api.CreateCommandPayload
		isValid     bool
	}{
		{
			description: "nil model",
			input:       nil,
			expected:    nil,
			isValid:     false,
		},
		{
			description: "template name only no parameters",
			input: &runCommandModel{
				CommandTemplateName: types.StringValue("RunShellScript"),
				Parameters:          types.MapNull(types.StringType),
			},
			expected: func() *v1api.CreateCommandPayload {
				p := v1api.NewCreateCommandPayload("RunShellScript")
				return p
			}(),
			isValid: true,
		},
		{
			description: "template name with parameters",
			input: &runCommandModel{
				CommandTemplateName: types.StringValue("RunShellScript"),
				Parameters: types.MapValueMust(types.StringType, map[string]attr.Value{
					"script": types.StringValue("echo hello"),
				}),
			},
			expected: func() *v1api.CreateCommandPayload {
				p := v1api.NewCreateCommandPayload("RunShellScript")
				p.SetParameters(map[string]string{"script": "echo hello"})
				return p
			}(),
			isValid: true,
		},
		{
			description: "template name with multiple parameters",
			input: &runCommandModel{
				CommandTemplateName: types.StringValue("RunPowerShellScript"),
				Parameters: types.MapValueMust(types.StringType, map[string]attr.Value{
					"script":  types.StringValue("Write-Output 'hello'"),
					"timeout": types.StringValue("30"),
				}),
			},
			expected: func() *v1api.CreateCommandPayload {
				p := v1api.NewCreateCommandPayload("RunPowerShellScript")
				p.SetParameters(map[string]string{
					"script":  "Write-Output 'hello'",
					"timeout": "30",
				})
				return p
			}(),
			isValid: true,
		},
		{
			description: "empty parameters map",
			input: &runCommandModel{
				CommandTemplateName: types.StringValue("RunShellScript"),
				Parameters:          types.MapValueMust(types.StringType, map[string]attr.Value{}),
			},
			expected: func() *v1api.CreateCommandPayload {
				p := v1api.NewCreateCommandPayload("RunShellScript")
				p.SetParameters(map[string]string{})
				return p
			}(),
			isValid: true,
		},
		{
			description: "empty template name",
			input: &runCommandModel{
				CommandTemplateName: types.StringValue(""),
				Parameters:          types.MapNull(types.StringType),
			},
			expected: func() *v1api.CreateCommandPayload {
				p := v1api.NewCreateCommandPayload("")
				return p
			}(),
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			ctx := context.TODO()
			output, err := toCreatePayload(ctx, tt.input)
			if !tt.isValid && err == nil {
				t.Fatalf("Should have failed")
			}
			if tt.isValid && err != nil {
				t.Fatalf("Should not have failed: %v", err)
			}
			if tt.isValid {
				diff := cmp.Diff(output, tt.expected,
					cmpopts.IgnoreUnexported(v1api.CreateCommandPayload{}),
				)
				if diff != "" {
					t.Fatalf("Data does not match: %s", diff)
				}
			}
		})
	}
}
