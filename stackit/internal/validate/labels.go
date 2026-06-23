package validate

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func LabelValidators() []validator.Map {
	return []validator.Map{
		mapvalidator.KeysAre(
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^.{1,63}$`),
				"must be between 1 and 63 characters long"),
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^[-A-Za-z0-9_.]*$`),
				"may only include alphanumerical characters, dashes, underscores and dots"),
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^([A-Za-z0-9].*)?[A-Za-z0-9]$`),
				"must begin and end with an alphanumerical character"),
		),
		mapvalidator.ValueStringsAre(
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^.{0,63}$`),
				"must not be longer than 63 characters"),
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^[-A-Za-z0-9_.]*$`),
				"may only include alphanumerical characters, dashes, underscores and dots"),
			stringvalidator.RegexMatches(
				regexp.MustCompile(`^(([A-Za-z0-9].*)?[A-Za-z0-9])?$`),
				"must begin and end with an alphanumerical character"),
		),
	}
}
