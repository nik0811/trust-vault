package main

import (
	"regexp"
	"strings"
)

// PIIPattern defines a PII detection pattern
type PIIPattern struct {
	Regex      *regexp.Regexp
	Confidence float64
	Validator  func(string) bool
}

var piiPatterns map[string]PIIPattern

func init() {
	piiPatterns = make(map[string]PIIPattern)

	patterns := map[string]struct {
		Pattern    string
		Confidence float64
		Validator  func(string) bool
	}{
		"EMAIL": {
			Pattern:    `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			Confidence: 0.95,
			Validator:  validateEmail,
		},
		"PHONE": {
			Pattern:    `(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}`,
			Confidence: 0.90,
			Validator:  validatePhone,
		},
		"SSN": {
			Pattern:    `\b(?!000|666|9\d{2})\d{3}[-\s]?(?!00)\d{2}[-\s]?(?!0000)\d{4}\b`,
			Confidence: 0.95,
			Validator:  validateSSN,
		},
		"CREDIT_CARD": {
			Pattern:    `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`,
			Confidence: 0.95,
			Validator:  validateCreditCard,
		},
		"CREDIT_CARD_FORMATTED": {
			Pattern:    `\b(?:4[0-9]{3}|5[1-5][0-9]{2}|3[47][0-9]{2}|6(?:011|5[0-9]{2}))[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}\b`,
			Confidence: 0.95,
			Validator:  validateCreditCard,
		},
		"IP_ADDRESS": {
			Pattern:    `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
			Confidence: 0.90,
			Validator:  validateIPAddress,
		},
		"DATE_OF_BIRTH": {
			Pattern:    `\b(?:0?[1-9]|1[0-2])[-/](?:0?[1-9]|[12][0-9]|3[01])[-/](?:19|20)\d{2}\b`,
			Confidence: 0.75,
			Validator:  nil,
		},
		"PASSPORT": {
			Pattern:    `\b[A-Z]{1,2}[0-9]{6,9}\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"DRIVER_LICENSE": {
			Pattern:    `\b[A-Z]{1,2}[0-9]{5,8}\b`,
			Confidence: 0.65,
			Validator:  nil,
		},
		"IBAN": {
			Pattern:    `\b[A-Z]{2}[0-9]{2}[A-Z0-9]{4}[0-9]{7}(?:[A-Z0-9]?){0,16}\b`,
			Confidence: 0.90,
			Validator:  validateIBAN,
		},
		"ROUTING_NUMBER": {
			Pattern:    `\b[0-9]{9}\b`,
			Confidence: 0.60,
			Validator:  validateRoutingNumber,
		},
		"MAC_ADDRESS": {
			Pattern:    `\b(?:[0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`,
			Confidence: 0.95,
			Validator:  nil,
		},
		"IPV6_ADDRESS": {
			Pattern:    `\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`,
			Confidence: 0.90,
			Validator:  nil,
		},
		"AWS_ACCESS_KEY": {
			Pattern:    `\bAKIA[0-9A-Z]{16}\b`,
			Confidence: 0.98,
			Validator:  nil,
		},
		"JWT_TOKEN": {
			Pattern:    `\beyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*\b`,
			Confidence: 0.95,
			Validator:  nil,
		},
		"VIN": {
			Pattern:    `\b[A-HJ-NPR-Z0-9]{17}\b`,
			Confidence: 0.85,
			Validator:  validateVIN,
		},
		"US_ZIP": {
			Pattern:    `\b[0-9]{5}(?:-[0-9]{4})?\b`,
			Confidence: 0.70,
			Validator:  nil,
		},
		"UK_POSTCODE": {
			Pattern:    `\b[A-Z]{1,2}[0-9][0-9A-Z]?\s?[0-9][A-Z]{2}\b`,
			Confidence: 0.85,
			Validator:  nil,
		},
	}

	for name, p := range patterns {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			continue
		}
		piiPatterns[name] = PIIPattern{
			Regex:      re,
			Confidence: p.Confidence,
			Validator:  p.Validator,
		}
	}
}

func runPatternMatching(text string, entityTypes []string, threshold float64) []Entity {
	results := make([]Entity, 0)

	for entityType, pattern := range piiPatterns {
		if len(entityTypes) > 0 && !containsIgnoreCase(entityTypes, entityType) {
			continue
		}
		if pattern.Regex == nil {
			continue
		}

		matches := pattern.Regex.FindAllStringIndex(text, -1)
		for _, m := range matches {
			value := text[m[0]:m[1]]
			confidence := pattern.Confidence

			if pattern.Validator != nil && !pattern.Validator(value) {
				confidence *= 0.5
			}

			if confidence < threshold {
				continue
			}

			// Normalize entity type (CREDIT_CARD_FORMATTED -> CREDIT_CARD)
			normalizedType := entityType
			if entityType == "CREDIT_CARD_FORMATTED" {
				normalizedType = "CREDIT_CARD"
			}

			results = append(results, Entity{
				Type:       normalizedType,
				Value:      value,
				Start:      m[0],
				End:        m[1],
				Confidence: confidence,
			})
		}
	}

	return deduplicateEntities(results)
}

func deduplicateEntities(entities []Entity) []Entity {
	if len(entities) == 0 {
		return entities
	}

	// Remove overlapping entities, keeping higher confidence
	result := make([]Entity, 0, len(entities))
	for _, e := range entities {
		overlaps := false
		for i, existing := range result {
			if e.Start < existing.End && e.End > existing.Start {
				overlaps = true
				if e.Confidence > existing.Confidence {
					result[i] = e
				}
				break
			}
		}
		if !overlaps {
			result = append(result, e)
		}
	}
	return result
}

func containsIgnoreCase(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
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
	if area == "000" || area == "666" || area[0] == '9' {
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

func validateRoutingNumber(s string) bool {
	if len(s) != 9 {
		return false
	}
	weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0
	for i, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		sum += int(c-'0') * weights[i]
	}
	return sum%10 == 0
}

func validateVIN(s string) bool {
	if len(s) != 17 {
		return false
	}
	for _, c := range s {
		if c == 'I' || c == 'O' || c == 'Q' {
			return false
		}
	}
	return true
}

func extractDigits(s string) string {
	var digits strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			digits.WriteRune(c)
		}
	}
	return digits.String()
}
