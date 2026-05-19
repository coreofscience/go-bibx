package utils

import (
	"fmt"
	"strings"
)

// ExtractDOI extracts the DOI from a given DOI string, removing any prefixes.
func ExtractDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	prefixes := []string{
		"https://doi.org/",
		"http://doi.org/",
		"https://dx.doi.org/",
		"http://dx.doi.org/",
		"doi:",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(doi), prefix) {
			return doi[len(prefix):]
		}
	}
	return doi
}

// InvertName inverts the name from "First Last" to "Last, First".
func InvertName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, ",") {
		return name
	}
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return name
	}
	lastName := parts[len(parts)-1]
	firstNames := parts[:len(parts)-1]
	return fmt.Sprintf("%s, %s", lastName, strings.Join(firstNames, " "))
}
