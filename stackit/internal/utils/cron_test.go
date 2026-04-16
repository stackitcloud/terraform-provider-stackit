package utils

import (
	"testing"
)

func TestSimplifyCronString(t *testing.T) {
	tests := []struct {
		description string
		input       string
		expected    string
	}{
		{
			"simple schedule",
			"0 0 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros",
			"00 00 * * *",
			"0 0 * * *",
		},
		{
			"schedule with leading zeros 2",
			"00 001 * * *",
			"0 1 * * *",
		},
		{
			"schedule with leading zeros 3",
			"00 0010 * * *",
			"0 10 * * *",
		},
		{
			"simple schedule with slash",
			"0 0/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash",
			"00 00/6 * * *",
			"0 0/6 * * *",
		},
		{
			"schedule with leading zeros and slash 2",
			"00 001/06 * * *",
			"0 1/6 * * *",
		},
		{
			"simple schedule with comma",
			"0 10,15 * * *",
			"0 10,15 * * *",
		},
		{
			"schedule with leading zeros and comma",
			"0 010,0015 * * *",
			"0 10,15 * * *",
		},
		{
			"simple schedule with comma and slash",
			"0 0-11/10 * * *",
			"0 0-11/10 * * *",
		},
		{
			"schedule with leading zeros, comma, and slash",
			"00 000-011/010 * * *",
			"0 0-11/10 * * *",
		},
		{
			"schedule with multiple spaces",
			"00   01  * *  *",
			"0 1 * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := SimplifyCronString(tt.input)
			if output != tt.expected {
				t.Fatalf("Data does not match: %s", output)
			}
		})
	}
}
