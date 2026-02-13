package utils

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetEnvStringOrDefault(t *testing.T) {
	const defaultValue = "default_value"
	tests := []struct {
		name         string
		val          types.String
		envVar       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "value is set - should return value",
			val:          types.StringValue("configured_value"),
			envVar:       "TEST_ENV_VAR",
			envValue:     "env_value",
			defaultValue: defaultValue,
			expected:     "configured_value",
		},
		{
			name:         "value is null, env var set - should return env value",
			val:          types.StringNull(),
			envVar:       "TEST_ENV_VAR",
			envValue:     "env_value",
			defaultValue: defaultValue,
			expected:     "env_value",
		},
		{
			name:         "value is unknown, env var set - should return env value",
			val:          types.StringUnknown(),
			envVar:       "TEST_ENV_VAR",
			envValue:     "env_value",
			defaultValue: defaultValue,
			expected:     "env_value",
		},
		{
			name:         "value is null, env var not set - should return default",
			val:          types.StringNull(),
			envVar:       "TEST_ENV_VAR",
			envValue:     "",
			defaultValue: defaultValue,
			expected:     defaultValue,
		},
		{
			name:         "value is unknown, env var not set - should return default",
			val:          types.StringUnknown(),
			envVar:       "TEST_ENV_VAR",
			envValue:     "",
			defaultValue: defaultValue,
			expected:     defaultValue,
		},
		{
			name:         "value is null, env var not set, empty default - should return empty string",
			val:          types.StringNull(),
			envVar:       "TEST_ENV_VAR",
			envValue:     "",
			defaultValue: "",
			expected:     "",
		},
		{
			name:         "value is empty string - should return empty string",
			val:          types.StringValue(""),
			envVar:       "TEST_ENV_VAR",
			envValue:     "env_value",
			defaultValue: defaultValue,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any previous value first
			err := os.Unsetenv(tt.envVar)
			if err != nil {
				t.Errorf("Failed to unset environment variable %s: %v", tt.envVar, err)
			}

			// Setup environment variable
			if tt.envValue != "" {
				err = os.Setenv(tt.envVar, tt.envValue)
				if err != nil {
					t.Errorf("Failed to set environment variable %s: %v", tt.envVar, err)
				}
				defer func() {
					err := os.Unsetenv(tt.envVar)
					if err != nil {
						t.Errorf("Failed to unset environment variable %s: %v", tt.envVar, err)
					}
				}()
			}

			result := GetEnvStringOrDefault(tt.val, tt.envVar, tt.defaultValue)

			if result != tt.expected {
				t.Errorf("GetEnvStringOrDefault() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetEnvBoolIfValueAbsent(t *testing.T) {
	tests := []struct {
		name     string
		val      types.Bool
		envVar   string
		envValue string
		expected bool
	}{
		{
			name:     "value is true - should return true",
			val:      types.BoolValue(true),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "false",
			expected: true,
		},
		{
			name:     "value is false - should return false",
			val:      types.BoolValue(false),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "true",
			expected: false,
		},
		{
			name:     "value is null, env var is 'true' - should return true",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "true",
			expected: true,
		},
		{
			name:     "value is null, env var is 'True' (case insensitive) - should return true",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "True",
			expected: true,
		},
		{
			name:     "value is null, env var is 'TRUE' (case insensitive) - should return true",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "TRUE",
			expected: true,
		},
		{
			name:     "value is null, env var is '1' - should return true",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "1",
			expected: true,
		},
		{
			name:     "value is null, env var is 'false' - should return false",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "false",
			expected: false,
		},
		{
			name:     "value is null, env var is 'False' - should return false",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "False",
			expected: false,
		},
		{
			name:     "value is null, env var is '0' - should return false",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "0",
			expected: false,
		},
		{
			name:     "value is null, env var is empty - should return false",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "",
			expected: false,
		},
		{
			name:     "value is null, env var is random string - should return false",
			val:      types.BoolNull(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "random",
			expected: false,
		},
		{
			name:     "value is unknown, env var is 'true' - should return true",
			val:      types.BoolUnknown(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "true",
			expected: true,
		},
		{
			name:     "value is unknown, env var is '1' - should return true",
			val:      types.BoolUnknown(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "1",
			expected: true,
		},
		{
			name:     "value is unknown, env var is 'false' - should return false",
			val:      types.BoolUnknown(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "false",
			expected: false,
		},
		{
			name:     "value is unknown, env var not set - should return false",
			val:      types.BoolUnknown(),
			envVar:   "TEST_BOOL_ENV_VAR",
			envValue: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any previous value first
			err := os.Unsetenv(tt.envVar)
			if err != nil {
				t.Errorf("Failed to unset environment variable %s: %v", tt.envVar, err)
			}

			// Setup environment variable
			if tt.envValue != "" {
				err = os.Setenv(tt.envVar, tt.envValue)
				if err != nil {
					t.Errorf("Failed to set environment variable %s: %v", tt.envVar, err)
				}
				defer func() {
					err := os.Unsetenv(tt.envVar)
					if err != nil {
						t.Errorf("Failed to unset environment variable %s: %v", tt.envVar, err)
					}
				}()
			}

			result := GetEnvBoolIfValueAbsent(tt.val, tt.envVar)

			if result != tt.expected {
				t.Errorf("GetEnvBoolIfValueAbsent() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
