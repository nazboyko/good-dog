package session

import "testing"

func TestAgeInWords(t *testing.T) {
	cases := map[string]string{
		// every real fixture shape
		"4Y/1M/1W":         "four years old",
		"11Y/2M/0W":        "eleven years old",
		"1Y/6M/3W":         "one year old",
		"0Y/2M/3W":         "two months old",
		"0Y/11M/1W":        "eleven months old",
		"8 years 1 month":  "eight years old",
		"3 years 2 months": "three years old",
		"10 years 1 month": "ten years old",
		// edges
		"0Y/1M/2W":  "one month old",
		"0Y/0M/3W":  "just a few weeks old",
		"14 years":  "14 years old",
		"5 months":  "five months old",
		"":          "",
		"unknown":   "",
		"adult dog": "",
		// an unfilled provider value must not become a claim
		"0Y/0M/0W": "",
		"0 years":  "",
		// a range has no single honest answer
		"1-2 years":          "",
		"6 months to 1 year": "",
	}
	for in, want := range cases {
		if got := AgeInWords(in); got != want {
			t.Errorf("AgeInWords(%q) = %q, want %q", in, got, want)
		}
	}
}
