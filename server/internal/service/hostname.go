package service

import "strings"

// CanonicalHostname normalizes an ASCII DNS hostname for storage and lookup.
// An empty result means the input is not a supported hostname. IDN/punycode is
// deliberately a separate product decision rather than an implicit transform.
func CanonicalHostname(raw string) string {
	hostname := strings.ToLower(strings.TrimSpace(raw))
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" || len(hostname) > 253 {
		return ""
	}

	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return ""
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return ""
			}
		}
	}
	return hostname
}
