// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"testing"

	"code.forgejo.org/go-chi/binding"
	"github.com/gobwas/glob"
)

func getGlobPatternErrorString(pattern string) string {
	// It would be unwise to rely on that glob
	// compilation errors don't ever change.
	if _, err := glob.Compile(pattern); err != nil {
		return err.Error()
	}
	return ""
}

var globValidationTestCases = []validationTestCase{
	{
		description: "Empty glob pattern",
		data: TestForm{
			GlobPattern: "",
		},
		expectedErrors: binding.Errors{},
	},
	{
		description: "Valid glob",
		data: TestForm{
			GlobPattern: "{master,release*}",
		},
		expectedErrors: binding.Errors{},
	},

	{
		description: "Invalid glob",
		data: TestForm{
			GlobPattern: "[a-",
		},
		expectedErrors: binding.Errors{
			binding.Error{
				FieldNames:     []string{"GlobPattern"},
				Classification: ErrGlobPattern,
				Message:        getGlobPatternErrorString("[a-"),
			},
		},
	},
}

func Test_GlobPatternValidation(t *testing.T) {
	AddBindingRules()

	for _, testCase := range globValidationTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			performValidationTest(t, testCase)
		})
	}
}
