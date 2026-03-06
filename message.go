package kwtsms

import (
	"regexp"
	"strings"
	"unicode"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// hiddenChars contains invisible characters that break SMS delivery.
var hiddenChars = map[rune]bool{
	'\u200B': true, // zero-width space
	'\u200C': true, // zero-width non-joiner
	'\u200D': true, // zero-width joiner
	'\u2060': true, // word joiner
	'\u00AD': true, // soft hyphen
	'\uFEFF': true, // BOM / zero-width no-break space
	'\uFFFC': true, // object replacement character
}

// directionalChars contains bidirectional text control codes.
var directionalChars = map[rune]bool{
	'\u200E': true, // left-to-right mark
	'\u200F': true, // right-to-left mark
	'\u202A': true, // LRE
	'\u202B': true, // RLE
	'\u202C': true, // PDF
	'\u202D': true, // LRO
	'\u202E': true, // RLO
	'\u2066': true, // LRI
	'\u2067': true, // RLI
	'\u2068': true, // FSI
	'\u2069': true, // PDI
}

// isEmoji returns true for emoji and pictographic codepoints that break SMS delivery.
func isEmoji(r rune) bool {
	cp := uint32(r)
	return (cp >= 0x1F600 && cp <= 0x1F64F) || // emoticons
		(cp >= 0x1F300 && cp <= 0x1F5FF) || // misc symbols and pictographs
		(cp >= 0x1F680 && cp <= 0x1F6FF) || // transport and map
		(cp >= 0x1F700 && cp <= 0x1F77F) || // alchemical symbols
		(cp >= 0x1F780 && cp <= 0x1F7FF) || // geometric shapes extended
		(cp >= 0x1F800 && cp <= 0x1F8FF) || // supplemental arrows-C
		(cp >= 0x1F900 && cp <= 0x1F9FF) || // supplemental symbols and pictographs
		(cp >= 0x1FA00 && cp <= 0x1FA6F) || // chess symbols
		(cp >= 0x1FA70 && cp <= 0x1FAFF) || // symbols and pictographs extended-A
		(cp >= 0x2600 && cp <= 0x26FF) || // miscellaneous symbols
		(cp >= 0x2700 && cp <= 0x27BF) || // dingbats
		(cp >= 0xFE00 && cp <= 0xFE0F) || // variation selectors
		(cp >= 0x1F000 && cp <= 0x1F0FF) || // mahjong tiles + playing cards
		(cp >= 0x1F1E0 && cp <= 0x1F1FF) || // regional indicator symbols (flags)
		cp == 0x20E3 || // combining enclosing keycap
		(cp >= 0xE0000 && cp <= 0xE007F) // tags block (subdivision flags)
}

// isC0C1Control returns true for C0/C1 control characters except \n and \t.
func isC0C1Control(r rune) bool {
	cp := uint32(r)
	if r == '\n' || r == '\t' {
		return false
	}
	return (cp <= 0x1F) || // C0 controls
		(cp == 0x7F) || // DEL
		(cp >= 0x80 && cp <= 0x9F) // C1 controls
}

// CleanMessage cleans SMS message text before sending to kwtSMS.
//
// Called automatically by Send(). No manual call needed.
//
// Strips content that silently breaks delivery:
//   - Arabic-Indic / Extended Arabic-Indic digits converted to Latin
//   - Emojis and pictographic symbols removed
//   - Hidden control characters (BOM, zero-width space, soft hyphen, etc.) removed
//   - Directional formatting characters removed
//   - C0/C1 control characters removed (preserves \n and \t)
//   - HTML tags stripped
//
// Arabic letters are NOT stripped. Arabic text is fully supported by kwtSMS.
func CleanMessage(text string) string {
	// 1. Convert Arabic-Indic and Extended Arabic-Indic digits to Latin
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if latin, ok := arabicToLatin[r]; ok {
			b.WriteRune(latin)
		} else {
			b.WriteRune(r)
		}
	}
	text = b.String()

	// 2. Strip HTML tags
	text = htmlTagRe.ReplaceAllString(text, "")

	// 3. Remove emojis, hidden chars, directional chars, and C0/C1 controls
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		if isEmoji(r) {
			continue
		}
		if hiddenChars[r] {
			continue
		}
		if directionalChars[r] {
			continue
		}
		if isC0C1Control(r) {
			continue
		}
		// Also check for Unicode Cc/Cf categories that aren't \n or \t
		if r != '\n' && r != '\t' && (unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Cf, r)) {
			continue
		}
		out.WriteRune(r)
	}

	return out.String()
}
