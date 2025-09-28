package validator_test

import (
	"chatapp-backend/internal/validator"
	"fmt"
	"testing"
)

func TestEmail(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectedError error
	}{
		// valid cases
		{
			name:          "Valid: Standard email",
			email:         "user@gmail.com",
			expectedError: nil,
		},
		{
			name:          "Valid: Email with plus sign in local part",
			email:         "user+tag@yahoo.co.uk",
			expectedError: nil,
		},
		{
			name:          "Valid: Email with underscore and dot in local part",
			email:         "first.last_name@yahoo.co.uk",
			expectedError: nil,
		},
		{
			name:          "Valid: Maximum length (64 chars)",
			email:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@protonmail.com",
			expectedError: nil,
		},

		// too long
		{
			name:          "Error: Too long (67 characters)",
			email:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@web.de",
			expectedError: fmt.Errorf("long_email"),
		},

		// bad format
		{
			name:          "Error: Missing @ sign",
			email:         "userexample.com",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Missing domain part",
			email:         "user@",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Missing TLD",
			email:         "user@domain",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Local part starting with dot",
			email:         ".user@example.com",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Local part ending with dot",
			email:         "user.@example.com",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Domain part starting with hyphen",
			email:         "user@-example.com",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: Domain part ending with hyphen",
			email:         "user@example-.com",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: TLD too short (1 character)",
			email:         "user@example.c",
			expectedError: fmt.Errorf("bad_format"),
		},
		{
			name:          "Error: TLD contains a number",
			email:         "user@example.c1",
			expectedError: fmt.Errorf("bad_format"),
		},

		// unknown domain
		{
			name:          "Error: Unsupported domain",
			email:         "user@wasistdas.com",
			expectedError: fmt.Errorf("unknown_domain"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Email(tc.email)

			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("Email(%q) failed unexpectedly: got error %v, want nil", tc.email, err)
				}
				return
			}

			if err == nil {
				t.Errorf("Email(%q) passed unexpectedly: got nil, want error %v", tc.email, tc.expectedError)
				return
			}

			if err.Error() != tc.expectedError.Error() {
				t.Errorf("Email(%q) got error %q, want error %q", tc.email, err.Error(), tc.expectedError.Error())
			}
		})
	}
}

func TestPassword(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		expectedError error
	}{
		{
			name:          "Valid Password: Minimum Length",
			password:      "aA1bB2",
			expectedError: nil,
		},
		{
			name:          "Valid Password: Maximum Length",
			password:      "aBc12345678901234567890123456789",
			expectedError: nil,
		},
		{
			name:          "Valid Password: Mixed Case and Symbols",
			password:      "P@sswOrd123!",
			expectedError: nil,
		},

		{
			name:          "Error: Password Too Short",
			password:      "aA1",
			expectedError: fmt.Errorf("short_password"),
		},
		{
			name:          "Error: Password Too Long",
			password:      "aBc123456789012345678901234567890123",
			expectedError: fmt.Errorf("long_password"),
		},

		{
			name:          "Error: Missing Lowercase Character",
			password:      "AABBCC1234",
			expectedError: fmt.Errorf("no_lowercase"),
		},
		{
			name:          "Error: Missing Uppercase Character",
			password:      "aabbcc1234",
			expectedError: fmt.Errorf("no_uppercase"),
		},
		{
			name:          "Error: Missing Number",
			password:      "PasswordABC",
			expectedError: fmt.Errorf("no_number"),
		},

		{
			name:          "Error: Multiple Violations - Missing Lowercase Expected",
			password:      "AAAABBBBCCCC",
			expectedError: fmt.Errorf("no_lowercase"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Password(tc.password)

			if tc.expectedError == nil {
				if err != nil {
					t.Errorf("Password(%q) failed unexpectedly: got error %v, want nil", tc.password, err)
				}
				return
			}

			if err == nil {
				t.Errorf("Password(%q) passed unexpectedly: got nil, want error %v", tc.password, tc.expectedError)
				return
			}

			if err.Error() != tc.expectedError.Error() {
				t.Errorf("Password(%q) got error %q, want error %q", tc.password, err.Error(), tc.expectedError.Error())
			}
		})
	}
}
