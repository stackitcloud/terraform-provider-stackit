package utils

import (
	"fmt"
	"regexp"
)

// ParseNameFromEmail extracts the name component from a service account email address.
// The expected email format is `name-<random7to10characters>@sa.stackit.cloud`
// or `name-<random7to10characters>@ske.sa.stackit.cloud`.
func ParseNameFromEmail(email string) (string, error) {
	namePattern := `^([a-z][a-z0-9]*(?:-[a-z0-9]+)*)-\w{7,10}@(?:ske\.)?sa\.stackit\.cloud$`
	re := regexp.MustCompile(namePattern)
	match := re.FindStringSubmatch(email)

	// If a match is found, return the name component
	if len(match) > 1 {
		return match[1], nil
	}

	// If no match is found, return an error
	return "", fmt.Errorf("unable to parse name from email")
}
