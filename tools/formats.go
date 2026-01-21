package tools

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// snakeLetters will match to the first letter and an underscore followed by a letter
var snakeLetters = regexp.MustCompile("(^[a-z])|_[a-z0-9]")

func ToPascalCase(in string) string {
	inputSplit := strings.Split(in, ".")

	var ucName string

	for _, v := range inputSplit {
		if len(v) < 1 {
			continue
		}

		firstChar := v[0:1]
		ucFirstChar := strings.ToUpper(firstChar)

		if len(v) < 2 {
			ucName += ucFirstChar
			continue
		}

		ucName += ucFirstChar + v[1:]
	}

	return snakeLetters.ReplaceAllStringFunc(ucName, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

func ToCamelCase(in string) string {
	pascal := ToPascalCase(in)

	// Grab first rune and lower case it
	firstLetter, size := utf8.DecodeRuneInString(pascal)
	if firstLetter == utf8.RuneError && size <= 1 {
		return pascal
	}

	return string(unicode.ToLower(firstLetter)) + pascal[size:]
}

func ValidateSnakeCase(in string) bool {
	return snakeLetters.MatchString(string(in))
}
