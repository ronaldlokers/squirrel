package web

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	aField     = regexp.MustCompile(`<(input|select|textarea)\b[^>]*>`)
	anAttrOnIt = regexp.MustCompile(`([a-zA-Z-]+)="([^"]*)"`)
	aLabelFor  = regexp.MustCompile(`<label[^>]*\bfor="([^"]+)"`)
	someWords  = regexp.MustCompile(`>\s*([^<>{}\s][^<>{}]*)<`)
)

func wrappedInALabelThatSaysSomething(body string, at int) bool {
	before := strings.LastIndex(body[:at], "<label")
	if before < 0 {
		return false
	}
	if shut := strings.LastIndex(body[:at], "</label>"); shut > before {
		return false
	}
	end := strings.Index(body[before:], "</label>")
	if end < 0 {
		return false
	}
	return someWords.MatchString(body[before : before+end])
}

func namedBy(tag string) map[string]string {
	out := map[string]string{}
	for _, a := range anAttrOnIt.FindAllStringSubmatch(tag, -1) {
		out[a[1]] = a[2]
	}
	return out
}

func TestEveryFieldYouCanTypeInHasAName(t *testing.T) {
	for name, body := range templates(t) {
		labelled := map[string]bool{}
		for _, m := range aLabelFor.FindAllStringSubmatch(body, -1) {
			labelled[m[1]] = true
		}
		for _, where := range aField.FindAllStringIndex(body, -1) {
			tag := body[where[0]:where[1]]
			at := namedBy(tag)
			if at["type"] == "hidden" || at["type"] == "submit" {
				continue
			}
			named := at["aria-label"] != "" ||
				at["aria-labelledby"] != "" ||
				at["placeholder"] != "" ||
				at["title"] != "" ||
				labelled[at["id"]] ||
				wrappedInALabelThatSaysSomething(body, where[0])
			require.True(t, named,
				"%s: a field with no accessible name — a screen reader reads it as an unnamed edit field:\n  %s",
				name, tag)
		}
	}
}
