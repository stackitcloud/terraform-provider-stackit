package utils

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	DisplayNameMinimumChars = 4
	DisplayNameMaximumChars = 8
	DescriptionMaxLength    = 256
	TokenMinDuration        = 600
	TokenMaxDuration        = 15552000
)

func CheckExpiration(expiresAt types.String, recreateBefore types.Int64, currentTime time.Time) (bool, error) {
	if expiresAt.IsNull() {
		return true, nil
	}

	if expiresAt.IsUnknown() {
		return true, nil
	}

	expiresAtTime, err := time.Parse(time.RFC3339, expiresAt.ValueString())
	if err != nil {
		return false, fmt.Errorf("failed to convert expiresAt field to timestamp: %w", err)
	}

	if !recreateBefore.IsNull() {
		expiresAtTime = expiresAtTime.Add(-time.Duration(recreateBefore.ValueInt64()) * time.Second)
	}

	// The value is considered expired if the expiration time is not after the current time.
	// This correctly handles cases where the expiration is before or exactly at the current time.
	if !expiresAtTime.After(currentTime) {
		return true, nil
	}

	return false, nil
}
