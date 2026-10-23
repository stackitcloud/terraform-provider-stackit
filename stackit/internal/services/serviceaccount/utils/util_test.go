package utils

import (
	"testing"
)

func TestParseNameFromEmail(t *testing.T) {
	testCases := []struct {
		email       string
		expected    string
		shouldError bool
	}{
		// Standard SA domain (Positive: 7 to 10 random characters)
		{"foo-vshp191@sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-8565oq12@sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-acfj2s123@sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@sa.stackit.cloud", "baz", false},        // 10 chars

		// Standard SA domain (Negative: 6 and 11 random characters)
		{"foo-vshp19@sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-8565oq12345@sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// SKE SA domain (Positive: 7 to 10 random characters)
		{"foo-qnmbwo1@ske.sa.stackit.cloud", "foo", false},           // 7 chars
		{"bar-qnmbwo12@ske.sa.stackit.cloud", "bar", false},          // 8 chars
		{"foo-bar-qnmbwo123@ske.sa.stackit.cloud", "foo-bar", false}, // 9 chars
		{"baz-abcdefghij@ske.sa.stackit.cloud", "baz", false},        // 10 chars

		// SKE SA domain (Negative: 6 and 11 random characters)
		{"foo-qnmbwo@ske.sa.stackit.cloud", "", true},      // 6 chars (Too short)
		{"bar-qnmbwo12345@ske.sa.stackit.cloud", "", true}, // 11 chars (Too long)

		// Invalid cases (Formatting & Unknown Domains)
		{"invalid-email@sa.stackit.cloud", "", true},
		{"missingcode-@sa.stackit.cloud", "", true},
		{"nohyphen8565oq1@sa.stackit.cloud", "", true},
		{"eu01-qnmbwo1@unknown.stackit.cloud", "", true},
		{"eu01-qnmbwo1@ske.stackit.com", "", true}, // Missing .sa. and ends in .com
		{"someotherformat@sa.stackit.cloud", "", true},
		{"invalid-format@ske.sa.stackit.cloud", "", true}, // SKE domain but missing the character suffix completely
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			name, err := ParseNameFromEmail(tc.email)
			if tc.shouldError {
				if err == nil {
					t.Errorf("expected an error for email: %s, but got none", tc.email)
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error for email: %s, but got: %v", tc.email, err)
				}
				if name != tc.expected {
					t.Errorf("expected name: %s, got: %s for email: %s", tc.expected, name, tc.email)
				}
			}
		})
	}
}
