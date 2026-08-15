package session

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// The reveal says the age in plain words. The listing's own string stays
// on the record for the transparency panel, this is only for the moment.

var (
	compactAge = regexp.MustCompile(`(?i)^\s*(\d+)\s*Y\s*/\s*(\d+)\s*M(?:\s*/\s*(\d+)\s*W)?\s*$`)
	proseAge   = regexp.MustCompile(`(?i)(\d+)\s*(year|month|week)s?`)
)

// AgeInWords turns a listing age like "4Y/1M/1W" or "8 years 1 month"
// into "four years old" or "two months old". Anything it cannot read,
// an all zero value, or a range comes back empty, and the reveal simply
// says nothing about age. A wrong number about a real dog is worse
// than silence.
func AgeInWords(ageText string) string {
	years, months, weeks, ok := parseAge(ageText)
	if !ok || years+months+weeks == 0 {
		return ""
	}
	switch {
	case years >= 1:
		return fmt.Sprintf("%s year%s old", numberWord(years), plural(years))
	case months >= 1:
		return fmt.Sprintf("%s month%s old", numberWord(months), plural(months))
	default:
		return "just a few weeks old"
	}
}

// a digit, optionally a unit word, then a range marker, then a digit:
// covers "1-2 years" and "6 months to 1 year" alike
var rangeMarker = regexp.MustCompile(`(?i)\d\s*(?:years?|months?|weeks?)?\s*(?:-|–|to)\s*\d`)

func parseAge(text string) (years, months, weeks int, ok bool) {
	// "1-2 years" has no single honest answer, say nothing
	if rangeMarker.MatchString(text) {
		return 0, 0, 0, false
	}
	if m := compactAge.FindStringSubmatch(text); m != nil {
		years, _ = strconv.Atoi(m[1])
		months, _ = strconv.Atoi(m[2])
		weeks, _ = strconv.Atoi(m[3])
		return years, months, weeks, true
	}
	found := false
	for _, m := range proseAge.FindAllStringSubmatch(text, -1) {
		n, _ := strconv.Atoi(m[1])
		switch strings.ToLower(m[2]) {
		case "year":
			years, found = n, true
		case "month":
			months, found = n, true
		case "week":
			weeks, found = n, true
		}
	}
	return years, months, weeks, found
}

var smallNumbers = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"}

func numberWord(n int) string {
	if n >= 0 && n < len(smallNumbers) {
		return smallNumbers[n]
	}
	return strconv.Itoa(n)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
