package assessment

import (
	"regexp"
	"strings"
)

// FormatType represents the detected format category of a cell value.
type FormatType int

const (
	FormatDate    FormatType = iota // highest priority
	FormatNumeric                   // second priority
	FormatBoolean                   // third priority
	FormatText                      // lowest priority (fallback)
)

// Pre-compiled regex patterns for format detection
var (
	// Date patterns
	// yyyy/MM/dd or yyyy-MM-dd
	dateISOPattern = regexp.MustCompile(`^\d{4}[/-](0?[1-9]|1[0-2])[/-](0?[1-9]|[12]\d|3[01])$`)
	// ROC date: yyy.M.d (e.g. 113.1.5, 112.12.31)
	dateROCPattern = regexp.MustCompile(`^\d{2,3}\.(0?[1-9]|1[0-2])\.(0?[1-9]|[12]\d|3[01])$`)

	// Numeric patterns
	// Integer (optional sign): 123, -456
	numericIntPattern = regexp.MustCompile(`^-?\d+$`)
	// Thousands-separated: 1,234 or 1,234,567
	numericThousandsPattern = regexp.MustCompile(`^-?\d{1,3}(,\d{3})+$`)
	// Decimal: 1.5, -3.14, .5
	numericDecimalPattern = regexp.MustCompile(`^-?(\d+\.\d*|\.\d+)$`)
	// Thousands with decimal: 1,234.56
	numericThousandsDecimalPattern = regexp.MustCompile(`^-?\d{1,3}(,\d{3})+\.\d+$`)
)

// Boolean values (case-insensitive)
var booleanValues = map[string]bool{
	"true":  true,
	"false": true,
	"是":    true,
	"否":    true,
	"y":     true,
	"n":     true,
	"yes":   true,
	"no":    true,
}

// DetectFormatType classifies a string value into a format type.
// Priority order: date > numeric > boolean > text
func DetectFormatType(value string) FormatType {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return FormatText
	}

	// Priority 1: Date
	if isDate(trimmed) {
		return FormatDate
	}

	// Priority 2: Numeric
	if isNumeric(trimmed) {
		return FormatNumeric
	}

	// Priority 3: Boolean
	if isBoolean(trimmed) {
		return FormatBoolean
	}

	// Priority 4: Text (fallback)
	return FormatText
}

func isDate(s string) bool {
	return dateISOPattern.MatchString(s) || dateROCPattern.MatchString(s)
}

func isNumeric(s string) bool {
	return numericIntPattern.MatchString(s) ||
		numericThousandsPattern.MatchString(s) ||
		numericDecimalPattern.MatchString(s) ||
		numericThousandsDecimalPattern.MatchString(s)
}

func isBoolean(s string) bool {
	_, ok := booleanValues[strings.ToLower(s)]
	return ok
}

// FormatTypeLabel returns the Chinese display label for a FormatType.
func FormatTypeLabel(ft FormatType) string {
	switch ft {
	case FormatDate:
		return "日期"
	case FormatNumeric:
		return "數字"
	case FormatBoolean:
		return "布林"
	default:
		return "文字"
	}
}
