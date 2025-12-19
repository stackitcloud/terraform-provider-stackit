// Copyright (c) STACKIT

package features

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
)

func TestBetaResourcesEnabled(t *testing.T) {
	tests := []struct {
		description string
		data        *core.ProviderData
		envSet      bool
		envValue    string
		expected    bool
		expectWarn  bool
	}{
		{
			description: "Feature flag enabled, env var not set",
			data: &core.ProviderData{
				EnableBetaResources: true,
			},
			expected: true,
		},
		{
			description: "Feature flag is disabled, env var not set",
			data: &core.ProviderData{
				EnableBetaResources: false,
			},
			expected: false,
		},
		{
			description: "Feature flag, Env var not set",
			data:        &core.ProviderData{},
			expected:    false,
		},
		{
			description: "Feature flag not set, Env var is true",
			data:        &core.ProviderData{},
			envSet:      true,
			envValue:    "true",
			expected:    true,
		},
		{
			description: "Feature flag not set, Env var is false",
			data:        &core.ProviderData{},
			envSet:      true,
			envValue:    "false",
			expected:    false,
		},
		{
			description: "Feature flag not set, Env var is empty",
			data:        &core.ProviderData{},
			envSet:      true,
			envValue:    "",
			expectWarn:  true,
			expected:    false,
		},
		{
			description: "Feature flag not set, Env var is gibberish",
			data:        &core.ProviderData{},
			envSet:      true,
			envValue:    "gibberish",
			expectWarn:  true,
			expected:    false,
		},
		{
			description: "Feature flag enabled, Env var is true",
			data: &core.ProviderData{
				EnableBetaResources: true,
			},
			envSet:   true,
			envValue: "true",
			expected: true,
		},
		{
			description: "Feature flag enabled, Env var is false",
			data: &core.ProviderData{
				EnableBetaResources: true,
			},
			envSet:   true,
			envValue: "false",
			expected: false,
		},
		{
			description: "Feature flag enabled, Env var is empty",
			data: &core.ProviderData{
				EnableBetaResources: true,
			},
			envSet:     true,
			envValue:   "",
			expectWarn: true,
			expected:   true,
		},
		{
			description: "Feature flag enabled, Env var is gibberish",
			data: &core.ProviderData{
				EnableBetaResources: true,
			},
			envSet:     true,
			envValue:   "gibberish",
			expectWarn: true,
			expected:   true,
		},
		{
			description: "Feature flag disabled, Env var is true",
			data: &core.ProviderData{
				EnableBetaResources: false,
			},
			envSet:   true,
			envValue: "true",
			expected: true,
		},
		{
			description: "Feature flag disabled, Env var is false",
			data: &core.ProviderData{
				EnableBetaResources: false,
			},
			envSet:   true,
			envValue: "false",
			expected: false,
		},
		{
			description: "Feature flag disabled, Env var is empty",
			data: &core.ProviderData{
				EnableBetaResources: false,
			},
			envSet:     true,
			envValue:   "",
			expectWarn: true,
			expected:   false,
		},
		{
			description: "Feature flag disabled, Env var is gibberish",
			data: &core.ProviderData{
				EnableBetaResources: false,
			},
			envSet:     true,
			envValue:   "gibberish",
			expectWarn: true,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if tt.envSet {
				t.Setenv("STACKIT_TF_ENABLE_BETA_RESOURCES", tt.envValue)
			}
			diags := diag.Diagnostics{}

			result := BetaResourcesEnabled(context.Background(), tt.data, &diags)
			if result != tt.expected {
				t.Fatalf("Expected %t, got %t", tt.expected, result)
			}

			if tt.expectWarn && diags.WarningsCount() == 0 {
				t.Fatalf("Expected warning, got none")
			}
			if !tt.expectWarn && diags.WarningsCount() > 0 {
				t.Fatalf("Expected no warning, got %d", diags.WarningsCount())
			}
		})
	}
}

func TestCheckBetaResourcesEnabled(t *testing.T) {
	tests := []struct {
		description string
		betaEnabled bool
		expectError bool
		expectWarn  bool
	}{
		{
			description: "Beta enabled, show warning",
			betaEnabled: true,
			expectWarn:  true,
		},
		{
			description: "Beta disabled, show error",
			betaEnabled: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var envValue string
			if tt.betaEnabled {
				envValue = "true"
			} else {
				envValue = "false"
			}
			t.Setenv("STACKIT_TF_ENABLE_BETA_RESOURCES", envValue)

			diags := diag.Diagnostics{}
			CheckBetaResourcesEnabled(context.Background(), &core.ProviderData{}, &diags, "stackit_test", "resource")

			if tt.expectError && diags.ErrorsCount() == 0 {
				t.Fatalf("Expected error, got none")
			}
			if !tt.expectError && diags.ErrorsCount() > 0 {
				t.Fatalf("Expected no error, got %d", diags.ErrorsCount())
			}

			if tt.expectWarn && diags.WarningsCount() == 0 {
				t.Fatalf("Expected warning, got none")
			}
			if !tt.expectWarn && diags.WarningsCount() > 0 {
				t.Fatalf("Expected no warning, got %d", diags.WarningsCount())
			}
		})
	}
}
