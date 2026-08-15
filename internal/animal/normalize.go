package animal

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// One normalization pipeline. Descriptions arrive as untrusted html,
// the rest of the system sees clean text only.

var (
	// unclosed blocks are eaten to end of input so their body never leaks
	scriptBlocks = regexp.MustCompile(`(?is)<script\b[^>]*>.*?(</script>|$)`)
	styleBlocks  = regexp.MustCompile(`(?is)<style\b[^>]*>.*?(</style>|$)`)
	blockBreaks  = regexp.MustCompile(`(?i)</?(p|div|li|ul|ol|br|h[1-6]|table|tr)[^>]*>`)
	anyTag       = regexp.MustCompile(`<[^>]*>`)
	// nbsp included: &nbsp; decodes to U+00A0 which \s does not match
	spaces = regexp.MustCompile(`[\s\x{00A0}]+`)
)

// SanitizeDescription strips html down to plain text. Block tags become
// sentence breaks on their own lines, so extracted facts keep their edges
// and the transparency panel can show the listing's paragraphs. The text
// is still untrusted and enters prompts only inside the untrusted text
// boundary.
func SanitizeDescription(raw string) string {
	text := scriptBlocks.ReplaceAllString(raw, " ")
	text = styleBlocks.ReplaceAllString(text, " ")
	text = blockBreaks.ReplaceAllString(text, "\n")
	text = anyTag.ReplaceAllString(text, " ")
	text = html.UnescapeString(text)
	// a second pass catches tags that entities were hiding, like &lt;b&gt;
	text = anyTag.ReplaceAllString(text, " ")

	var sentences []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(spaces.ReplaceAllString(line, " "))
		if line == "" {
			continue
		}
		if last, _ := utf8.DecodeLastRuneInString(line); !strings.ContainsRune(".!?", last) {
			line += "."
		}
		sentences = append(sentences, line)
	}
	return strings.Join(sentences, "\n")
}

const minDescriptionChars = 200

// PoolFilter decides if a dog can enter the playable pool, with a
// human readable reason when it cannot.
func PoolFilter(a Animal) (bool, string) {
	if n := utf8.RuneCountInString(a.Description); n < minDescriptionChars {
		return false, fmt.Sprintf("description is %d chars, need %d", n, minDescriptionChars)
	}
	if a.PhotoURL == "" && a.PhotoLocal == "" {
		return false, "no photo"
	}
	if a.OrgID == "" {
		return false, "no organization"
	}
	return true, ""
}
