package validator

import (
	"fmt"
	"regexp"
	"strings"
)

func Email(email string) error {
	const maxlength = 64

	if len(email) > maxlength {
		return fmt.Errorf("long_email")
	}

	const emailRegex = `^[a-zA-Z0-9]([a-zA-Z0-9._+-]*[a-zA-Z0-9])?@[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?\.[a-zA-Z]{2,}$`
	if !regexp.MustCompile(emailRegex).MatchString(email) {
		return fmt.Errorf("bad_format")
	}

	for i := range len(domains) {
		if strings.HasSuffix(email, domains[i]) {
			return nil
		}
	}

	return fmt.Errorf("unknown_domain")
}

func Password(password string) error {
	length := len(password)
	if length < 6 {
		return fmt.Errorf("short_password")
	} else if length > 32 {
		return fmt.Errorf("long_password")
	}

	lowercase := regexp.MustCompile(`[a-z]`)
	uppercase := regexp.MustCompile(`[A-Z]`)
	number := regexp.MustCompile(`\d`)

	if !lowercase.MatchString(password) {
		return fmt.Errorf("no_lowercase")
	}
	if !uppercase.MatchString(password) {
		return fmt.Errorf("no_uppercase")
	}
	if !number.MatchString(password) {
		return fmt.Errorf("no_number")
	}
	return nil
}
