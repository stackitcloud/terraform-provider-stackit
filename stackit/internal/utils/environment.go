package utils

import (
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GetEnvStringOrDefault takes a Framework StringValue and a corresponding Environment Variable name and returns
// either the string value set in the StringValue if not Null / Unknown _or_ the os.GetEnv() value of the Environment
// Variable provided. If both of these are empty, an empty string defaultValue is returned.
func GetEnvStringOrDefault(val types.String, envVar, defaultValue string) string {
	if val.IsNull() || val.IsUnknown() {
		if v := os.Getenv(envVar); v != "" {
			return os.Getenv(envVar)
		}
		return defaultValue
	}

	return val.ValueString()
}

// GetEnvBoolIfValueAbsent takes a Framework BoolValue and a corresponding Environment Variable name and returns
// one of the following in priority order:
// 1 - the Boolean value set in the BoolValue if this is not Null / Unknown.
// 2 - the boolean representation of the os.GetEnv() value of the Environment Variable provided (where anything but
// 'true' or '1' is 'false').
// 3 - `false` in all other cases.
func GetEnvBoolIfValueAbsent(val types.Bool, envVar string) bool {
	if val.IsNull() || val.IsUnknown() {
		v := os.Getenv(envVar)
		if strings.EqualFold(v, "true") || strings.EqualFold(v, "1") {
			return true
		}
	}

	return val.ValueBool()
}
