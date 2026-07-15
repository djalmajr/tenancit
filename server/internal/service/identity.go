package service

import (
	"regexp"
	"strings"
)

var (
	tenantSlugPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	definitionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	integerValuePattern  = regexp.MustCompile(`^-?\d+$`)
)

// NormalizeTenantSlug validates the stable tenant identity used by consumer
// requests. It deliberately rejects implicit case conversion so callers do not
// accidentally create a different identity from what an operator entered.
func NormalizeTenantSlug(raw string) (string, bool) {
	slug := strings.TrimSpace(raw)
	return slug, len(slug) <= 63 && tenantSlugPattern.MatchString(slug)
}

// NormalizeDefinitionKey validates definition and field keys. These values are
// public contract identifiers, not display labels, so their character set is
// intentionally narrow and URL/JSON friendly.
func NormalizeDefinitionKey(raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	return key, definitionKeyPattern.MatchString(key)
}

func ValidResourceValue(dataType, value string) bool {
	if value == "" {
		return true
	}
	switch dataType {
	case "int":
		return integerValuePattern.MatchString(value)
	case "bool":
		return value == "true" || value == "false"
	default:
		return true
	}
}
