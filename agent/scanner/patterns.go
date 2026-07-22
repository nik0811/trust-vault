package scanner

import (
	"regexp"
	"strings"
)

type PIIPattern struct {
	Name       string
	Regex      *regexp.Regexp
	Confidence float64
	Severity   string
	Validator  func(string) bool
}

var PIIPatterns []PIIPattern

func init() {
	patterns := []struct {
		Name       string
		Pattern    string
		Confidence float64
		Severity   string
		Validator  func(string) bool
	}{
		{
			Name:       "EMAIL",
			Pattern:    `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Confidence: 0.95,
			Severity:   "MEDIUM",
			Validator:  validateEmail,
		},
		{
			Name:       "PHONE",
			Pattern:    `(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}`,
			Confidence: 0.90,
			Severity:   "MEDIUM",
			Validator:  validatePhone,
		},
		{
			Name:       "SSN",
			Pattern:    `\b(?!000|666|9\d{2})\d{3}[-\s]?(?!00)\d{2}[-\s]?(?!0000)\d{4}\b`,
			Confidence: 0.95,
			Severity:   "CRITICAL",
			Validator:  validateSSN,
		},
		{
			Name:       "CREDIT_CARD",
			Pattern:    `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`,
			Confidence: 0.95,
			Severity:   "CRITICAL",
			Validator:  validateCreditCard,
		},
		{
			Name:       "CREDIT_CARD_FORMATTED",
			Pattern:    `\b(?:4[0-9]{3}|5[1-5][0-9]{2}|3[47][0-9]{2}|6(?:011|5[0-9]{2}))[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}\b`,
			Confidence: 0.95,
			Severity:   "CRITICAL",
			Validator:  validateCreditCard,
		},
		{
			Name:       "IP_ADDRESS",
			Pattern:    `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
			Confidence: 0.90,
			Severity:   "LOW",
			Validator:  validateIPAddress,
		},
		{
			Name:       "AWS_ACCESS_KEY",
			Pattern:    `\bAKIA[0-9A-Z]{16}\b`,
			Confidence: 0.98,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "AWS_SECRET_KEY",
			Pattern:    `(?:aws_secret_access_key|secret_access_key|aws_secret)[=:\s]+['"]?([A-Za-z0-9/+=]{40})['"]?`,
			Confidence: 0.95,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "API_KEY",
			Pattern:    `(?:api[_-]?key|apikey|api_secret|secret_key|access_token)[=:\s]+['"]?([a-zA-Z0-9_-]{20,})['"]?`,
			Confidence: 0.85,
			Severity:   "HIGH",
			Validator:  nil,
		},
		{
			Name:       "JWT_TOKEN",
			Pattern:    `\beyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\b`,
			Confidence: 0.95,
			Severity:   "HIGH",
			Validator:  nil,
		},
		{
			Name:       "PRIVATE_KEY",
			Pattern:    `-----BEGIN (?:RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
			Confidence: 0.99,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "PASSPORT",
			Pattern:    `\b[A-Z]{1,2}[0-9]{6,9}\b`,
			Confidence: 0.70,
			Severity:   "HIGH",
			Validator:  nil,
		},
		{
			Name:       "DRIVER_LICENSE",
			Pattern:    `\b[A-Z]{1,2}[0-9]{5,8}\b`,
			Confidence: 0.65,
			Severity:   "HIGH",
			Validator:  nil,
		},
		{
			Name:       "IBAN",
			Pattern:    `\b[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}(?:[A-Z0-9]?){0,16}\b`,
			Confidence: 0.90,
			Severity:   "HIGH",
			Validator:  validateIBAN,
		},
		{
			Name:       "SWIFT_CODE",
			Pattern:    `\b[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}(?:[A-Z0-9]{3})?\b`,
			Confidence: 0.75,
			Severity:   "MEDIUM",
			Validator:  validateSWIFT,
		},
		{
			Name:       "MAC_ADDRESS",
			Pattern:    `\b(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`,
			Confidence: 0.95,
			Severity:   "LOW",
			Validator:  nil,
		},
		{
			Name:       "IPV6_ADDRESS",
			Pattern:    `\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`,
			Confidence: 0.90,
			Severity:   "LOW",
			Validator:  nil,
		},
		{
			Name:       "GITHUB_TOKEN",
			Pattern:    `\b(ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})\b`,
			Confidence: 0.98,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "SLACK_TOKEN",
			Pattern:    `\bxox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*\b`,
			Confidence: 0.98,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "STRIPE_KEY",
			Pattern:    `\b(sk_live_[a-zA-Z0-9]{24,}|pk_live_[a-zA-Z0-9]{24,})\b`,
			Confidence: 0.98,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
		{
			Name:       "DATABASE_URL",
			Pattern:    `(?:postgres|mysql|mongodb|redis)://[^\s'"]+`,
			Confidence: 0.95,
			Severity:   "CRITICAL",
			Validator:  nil,
		},
	}

	for _, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue
		}
		PIIPatterns = append(PIIPatterns, PIIPattern{
			Name:       p.Name,
			Regex:      re,
			Confidence: p.Confidence,
			Severity:   p.Severity,
			Validator:  p.Validator,
		})
	}
}

func validateEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

func validatePhone(s string) bool {
	digits := extractDigits(s)
	return len(digits) >= 10 && len(digits) <= 15
}

func validateSSN(s string) bool {
	digits := extractDigits(s)
	if len(digits) != 9 {
		return false
	}
	area := digits[0:3]
	if area == "000" || area == "666" || digits[0] == '9' {
		return false
	}
	return digits[3:5] != "00" && digits[5:9] != "0000"
}

func validateCreditCard(s string) bool {
	digits := extractDigits(s)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	return luhnCheck(digits)
}

func validateIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
			n = n*10 + int(c-'0')
		}
		if n > 255 {
			return false
		}
	}
	return true
}

func validateIBAN(s string) bool {
	return len(s) >= 15 && len(s) <= 34
}

func validateSWIFT(s string) bool {
	if len(s) != 8 && len(s) != 11 {
		return false
	}
	commonWords := []string{"CRITICAL", "PASSPORT", "REDACTED", "PASSWORD", "USERNAME", "INTERNAL", "EXTERNAL", "DOCUMENT", "CUSTOMER", "SECURITY"}
	for _, word := range commonWords {
		if s == word {
			return false
		}
	}
	return true
}

func extractDigits(s string) string {
	var result strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

func luhnCheck(digits string) bool {
	sum := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
