package main

import "testing"

func TestNamedInSurvivesHowSheltersWriteNames(t *testing.T) {
	pages := map[string]string{
		"plain":                 "<h1>Venus</h1>",
		"shouty":                "<title>VENUS D9548</title>",
		"wrapped in markup":     "<h1 class=\"name\">Sugar<span> </span>Bear</h1>",
		"nbsp between words":    "<h1>Sugar&nbsp;Bear</h1>",
		"newline between words": "<h1>Sugar\n  Bear</h1>",
	}
	for kind, body := range pages {
		name := "Venus"
		if kind != "plain" && kind != "shouty" {
			name = "Sugar Bear"
		}
		if !namedIn(name, body) {
			t.Errorf("%s: %q should be found in %q", kind, name, body)
		}
	}
}

func TestNamedInFailsWhenTheDogIsGone(t *testing.T) {
	adopted := "<h1>Sorry, this animal is no longer available</h1><p>See our other dogs.</p>"
	if namedIn("Venus", adopted) {
		t.Error("a page that never names the dog must not pass")
	}
	if namedIn("Sugar Bear", "<h1>Sugar</h1><p>a different dog</p>") {
		t.Error("half the name is not the name")
	}
}
