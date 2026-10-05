package utils

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCheckExpiration(t *testing.T) {
	// Reference time for testing
	now := time.Date(2025, 10, 26, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		expiresAt       types.String
		recreateBefore  types.Int64
		currentTime     time.Time
		expectedExpired bool
		expectedErr     bool
	}{
		{
			name:            "Is not expired",
			expiresAt:       types.StringValue(now.Add(1 * time.Hour).Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: false,
			expectedErr:     false,
		},
		{
			name:            "Is expired",
			expiresAt:       types.StringValue(now.Add(-1 * time.Hour).Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
		{
			name:            "Expires at the exact current time",
			expiresAt:       types.StringValue(now.Format(time.RFC3339)),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true, // Should be considered expired if the times are equal.
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt is null",
			expiresAt:       types.StringNull(),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt is unknown",
			expiresAt:       types.StringUnknown(),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: true, // Should be treated as expired to force re-creation
			expectedErr:     false,
		},
		{
			name:            "ExpiresAt has invalid format",
			expiresAt:       types.StringValue("invalid-time-format"),
			recreateBefore:  types.Int64Null(),
			currentTime:     now,
			expectedExpired: false,
			expectedErr:     true,
		},
		{
			name:            "Is considered expired due to recreateBefore",
			expiresAt:       types.StringValue(now.Add(30 * time.Minute).Format(time.RFC3339)),
			recreateBefore:  types.Int64Value(3600),
			currentTime:     now,
			expectedExpired: true,
			expectedErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasExpired, err := CheckExpiration(tt.expiresAt, tt.recreateBefore, tt.currentTime)

			if (err != nil) != tt.expectedErr {
				t.Errorf("CheckExpiration() error = %v, wantErr %v", err, tt.expectedErr)
				return
			}
			if hasExpired != tt.expectedExpired {
				t.Errorf("CheckExpiration() = %v, want %v", hasExpired, tt.expectedExpired)
			}
		})
	}
}
