package naming

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^0-9A-Za-z]+`)
	reserved    = map[string]struct{}{
		"abstract": {}, "as": {}, "base": {}, "bool": {}, "break": {}, "byte": {}, "case": {},
		"catch": {}, "char": {}, "checked": {}, "class": {}, "const": {}, "continue": {},
		"decimal": {}, "default": {}, "delegate": {}, "do": {}, "double": {}, "else": {},
		"enum": {}, "event": {}, "explicit": {}, "extern": {}, "false": {}, "finally": {},
		"fixed": {}, "float": {}, "for": {}, "foreach": {}, "goto": {}, "if": {}, "implicit": {},
		"in": {}, "int": {}, "interface": {}, "internal": {}, "is": {}, "lock": {}, "long": {},
		"namespace": {}, "new": {}, "null": {}, "object": {}, "operator": {}, "out": {}, "override": {},
		"params": {}, "private": {}, "protected": {}, "public": {}, "readonly": {}, "ref": {},
		"return": {}, "sbyte": {}, "sealed": {}, "short": {}, "sizeof": {}, "stackalloc": {},
		"static": {}, "string": {}, "struct": {}, "switch": {}, "this": {}, "throw": {}, "true": {},
		"try": {}, "typeof": {}, "uint": {}, "ulong": {}, "unchecked": {}, "unsafe": {}, "ushort": {},
		"using": {}, "virtual": {}, "void": {}, "volatile": {}, "while": {},
	}
)

// PascalCase converts any string into PascalCase.
func PascalCase(input string) string {
	parts := splitIntoWords(input)
	if len(parts) == 0 {
		return "Value"
	}

	var builder strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			builder.WriteString(strings.ToLower(part[1:]))
		}
	}

	return sanitizeLeadingCharacter(builder.String())
}

// CamelCase converts any string into camelCase.
func CamelCase(input string) string {
	pascal := PascalCase(input)
	if pascal == "" {
		return "value"
	}
	runes := []rune(pascal)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

// Identifier returns a camelCase identifier that avoids reserved keywords.
func Identifier(input string) string {
	id := CamelCase(input)
	return ensureNotReserved(id)
}

// PascalIdentifier ensures PascalCase output is safe for use as public identifiers.
func PascalIdentifier(input string) string {
	name := PascalCase(input)
	return ensureNotReserved(name)
}

func splitIntoWords(input string) []string {
	clean := nonAlphaNum.ReplaceAllString(input, " ")
	fields := strings.Fields(clean)
	if len(fields) == 0 && input != "" {
		return []string{input}
	}

	var result []string
	for _, field := range fields {
		result = append(result, splitCamel(field)...)
	}
	return result
}

func sanitizeLeadingCharacter(input string) string {
	if input == "" {
		return "Value"
	}
	runes := []rune(input)
	if unicode.IsDigit(runes[0]) {
		return "_" + input
	}
	return input
}

func ensureNotReserved(name string) string {
	if name == "" {
		return name
	}
	if _, found := reserved[strings.ToLower(name)]; found {
		return name + "Value"
	}
	return sanitizeLeadingCharacter(name)
}

func splitCamel(word string) []string {
	if word == "" {
		return nil
	}
	runes := []rune(word)
	if len(runes) == 1 {
		return []string{word}
	}

	start := 0
	var parts []string
	for i := 1; i < len(runes); i++ {
		prev := runes[i-1]
		curr := runes[i]
		var next rune
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		if shouldSplit(prev, curr, next) {
			parts = append(parts, string(runes[start:i]))
			start = i
		}
	}
	parts = append(parts, string(runes[start:]))
	return parts
}

func shouldSplit(prev, curr, next rune) bool {
	if unicode.IsDigit(curr) && !unicode.IsDigit(prev) {
		return true
	}
	if unicode.IsUpper(curr) && !unicode.IsUpper(prev) {
		return true
	}
	if unicode.IsUpper(curr) && unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next) {
		return true
	}
	return false
}
