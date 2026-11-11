package handlers

import (
	"testing"
)

func TestNormalizeEmail(test *testing.T) {
	for _, testCase := range []struct {
		input, out string
	}{
		{" User@Example.COM ", "user@example.com"},
		{"bad", ""},
		{"Name <user@example.com>", ""},
		{"", ""},
		{"a@b.com", "a@b.com"},
	} {
		got, isValidEmail := normalizeEmail(testCase.input)
		if got != testCase.out || isValidEmail != (testCase.out != "") {
			test.Errorf("%q: %q %v", testCase.input, got, isValidEmail)
		}
	}
}
