// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"regexp"
	"testing"

	"code.forgejo.org/go-chi/binding"
)

func getRegexPatternErrorString(pattern string) string {
	if _, err := regexp.Compile(pattern); err != nil {
		return err.Error()
	}
	return ""
}

var regexValidationTestCases = []validationTestCase{
	{
		description: "Empty regex pattern",
		data: TestForm{
			RegexPattern: "",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "Valid regex",
		data: TestForm{
			RegexPattern: `(\d{1,3})+`,
		},
		expectedErrors: binding.Errors{},
	},

	{
		description: "Invalid regex",
		data: TestForm{
			RegexPattern: "[a-",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"RegexPattern"},
				Classification: ErrRegexPattern,
				Message:        getRegexPatternErrorString("[a-"),
			},
		},
	},
}

func Test_RegexPatternValidation(t *testing.T) {
	AddBindingRules()

	for _, testCase := range regexValidationTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			performValidationTest(t, testCase)
		})
	}
}
