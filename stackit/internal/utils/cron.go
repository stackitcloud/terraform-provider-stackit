package utils

import (
	"regexp"
	"strings"
)

// SimplifyCronString removes leading 0s from backup schedule numbers (e.g. "00 00 * * *" becomes "0 0 * * *")
// Needed as some API might do it internally and would otherwise cause inconsistent result in Terraform
func SimplifyCronString(cron string) string {
	regex := regexp.MustCompile(`0+\d+`) // Matches series of one or more zeros followed by a series of one or more digits
	simplifiedCron := regex.ReplaceAllStringFunc(cron, func(match string) string {
		simplified := strings.TrimLeft(match, "0")
		if simplified == "" {
			simplified = "0"
		}
		return simplified
	})

	whiteSpaceRegex := regexp.MustCompile(`\s+`)
	simplifiedCron = whiteSpaceRegex.ReplaceAllString(simplifiedCron, " ")

	return simplifiedCron
}
