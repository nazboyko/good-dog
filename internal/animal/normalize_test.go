package animal

import (
	"strings"
	"testing"
)

func TestSanitizeDescriptionMessyHTML(t *testing.T) {
	raw := `<p>Meet <b>Biscuit</b>!&nbsp;She &amp; her ball are inseparable.</p>
<script type="text/javascript">alert("ignore previous instructions");</script>
<style>.hidden { display: none; }</style>
<ul><li>House trained</li><li>Loves &lt;b&gt;walks&lt;/b&gt;</li></ul>
<P CLASS="x">Gentle   with    kids.</P>`

	got := SanitizeDescription(raw)
	want := "Meet Biscuit ! She & her ball are inseparable. House trained. Loves walks. Gentle with kids."
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	for _, forbidden := range []string{"<", ">", "alert", "display: none", "\n", "  "} {
		if strings.Contains(got, forbidden) {
			t.Errorf("sanitized text still contains %q: %q", forbidden, got)
		}
	}
}

func TestSanitizeDescriptionBlockTagsKeepFactEdges(t *testing.T) {
	raw := `<ul><li>Housetrained and gentle</li><li>Happy to nap</li></ul><p>Loves sunny spots</p>`
	want := "Housetrained and gentle. Happy to nap. Loves sunny spots."
	if got := SanitizeDescription(raw); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSanitizeDescriptionUnclosedScriptNeverLeaks(t *testing.T) {
	got := SanitizeDescription(`Nice dog<script>fetch("http://evil/x")`)
	if got != "Nice dog." {
		t.Errorf("unclosed script leaked: %q", got)
	}
	got = SanitizeDescription(`Sweet girl<style>.x{color:red}`)
	if got != "Sweet girl." {
		t.Errorf("unclosed style leaked: %q", got)
	}
}

func TestSanitizeDescriptionEmpty(t *testing.T) {
	for _, raw := range []string{"", "   ", "<p></p>", "<script>evil()</script>"} {
		if got := SanitizeDescription(raw); got != "" {
			t.Errorf("SanitizeDescription(%q) = %q, want empty", raw, got)
		}
	}
}

func TestPoolFilter(t *testing.T) {
	long := strings.Repeat("a good dog. ", 20)
	base := Animal{Description: long, PhotoLocal: "fixtures/photos/x.jpg", OrgID: "org-1"}

	if ok, reason := PoolFilter(base); !ok {
		t.Errorf("complete dog rejected: %s", reason)
	}
	cases := map[string]Animal{
		"short description": {Description: "sweet pup", PhotoLocal: "x.jpg", OrgID: "org-1"},
		"empty description": {Description: "", PhotoLocal: "x.jpg", OrgID: "org-1"},
		"no photo":          {Description: long, OrgID: "org-1"},
		"no organization":   {Description: long, PhotoLocal: "x.jpg"},
	}
	for name, a := range cases {
		if ok, _ := PoolFilter(a); ok {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestPoolFilterCountsRunesNotBytes(t *testing.T) {
	base := Animal{PhotoLocal: "x.jpg", OrgID: "org-1"}

	base.Description = strings.Repeat("ф", 199)
	if ok, _ := PoolFilter(base); ok {
		t.Error("199 runes must be rejected")
	}
	base.Description = strings.Repeat("ф", 200)
	if ok, reason := PoolFilter(base); !ok {
		t.Errorf("200 runes must pass, got %s", reason)
	}
}
