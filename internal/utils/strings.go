package utils

import "strings"

// KeepLongestString returns the longest of two strings.
func KeepLongestString(a, b string) string {
	if len(a) > len(b) {
		return a
	}
	if len(b) > len(a) {
		return b
	}
	if a > b {
		return b
	}
	return a
}

// InvertAbstract returns the string representation of an inverted abstract.
func InvertAbstract(abstractInvertedIndex *map[string][]int) string {
	if abstractInvertedIndex == nil {
		return ""
	}
	length := 0
	for _, indices := range *abstractInvertedIndex {
		maxIndex := 0
		for _, index := range indices {
			if index > maxIndex {
				maxIndex = index
			}
		}
		if maxIndex > length {
			length = maxIndex
		}
	}
	words := make([]string, length+1)
	for word, indices := range *abstractInvertedIndex {
		for _, index := range indices {
			words[index] = word
		}
	}
	return strings.TrimSpace(strings.Join(words, " "))
}

// WrapText wraps the given text to a maximum line length of limit.
func WrapText(text string, limit int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}
	var sb strings.Builder
	lineLen := 0
	for i, word := range words {
		if lineLen+len(word)+1 > limit && lineLen > 0 {
			sb.WriteString("\n")
			lineLen = 0
		} else if i > 0 {
			sb.WriteString(" ")
			lineLen++
		}
		sb.WriteString(word)
		lineLen += len(word)
	}
	return sb.String()
}
