package builtin

import (
	"regexp"
	"strings"
)

var GroupNames = map[string]struct{}{
	"email":             {},
	"email_localpart":   {},
	"email_domain":      {},
	"phone":             {},
	"phone_countrycode": {},
	"phone_areacode":    {},
	"creditcard":        {},
	"url":               {},
	"domain":            {},
}

const domainRegex = `(?P<DOMAIN>[A-Za-z\d](?:[A-Za-z\d\-.]*[A-Za-z\d])?\.[A-Za-z\d]{1,63})`
const emailRegex = `(?P<EMAIL>(?P<EMAIL_LOCALPART>(?:(?:[A-Za-z\d_%+])(?:[A-Za-z\d_%+\-.]*(?:[A-Za-z\d_%+]))?){1,64})@(?P<EMAIL_DOMAIN>([A-Za-z\d](?:[A-Za-z\d\-.]*[A-Za-z\d])?\.[A-Za-z\d]{1,63})))`
const phoneRegex = `(?P<PHONE>\+(?P<PHONE_COUNTRYCODE>\d{1,3})\s?(?P<PHONE_AREACODE>(?:\d{3}|\(\d{3}\)))[\d .\-]{0,13}\d)`
const creditCardRegex = `(?P<CREDITCARD>(?:\d{4}(?:\s|-)?){4})`
const urlRegex = `(?P<URL>[A-Za-z][A-Za-z\d+\-.]*(?:://)([A-Za-z\d](?:[A-Za-z\d\-.]*[A-Za-z\d])?\.[A-Za-z\d]{1,63})(?::\d+)?(?:/[A-Za-z\d_%/\-.~]*)?(?:\?[A-Za-z\d_=&%+\-.~]*)?(?:#[A-Za-z\d_%\-.~/?:@!$&'()+,;=]*)?)`

var Builtins = map[string]string{ // regex pulled from ./*.lx
	"domain_unwrapped": unwrap(domainRegex),
	"domain":           format(domainRegex),
	"email":            format(emailRegex),
	"phone":            format(phoneRegex),
	"creditcard":       format(creditCardRegex),
	"url":              format(urlRegex),
}

var captureRegex = regexp.MustCompile(`\(\?P<\w+>`)

func format(regex string) string {
	if !strings.HasPrefix(regex, "(?P<") {
		return regex
	}
	new := captureRegex.ReplaceAllStringFunc(regex, func(match string) string {
		parts := strings.Split(match, "<")
		return parts[0] + "<" + strings.ToLower(parts[1])
	})
	return new
}

func unwrap(regex string) string {
	if strings.HasPrefix(regex, "(?P<") {
		regex = captureRegex.ReplaceAllString(regex, "")
		return "(" + regex
	}
	return regex
}
