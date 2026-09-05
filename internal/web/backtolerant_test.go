package web

import "testing"

func TestBackTolerantRejectsEveryWayOutOfTheHost(t *testing.T) {
	cases := []struct {
		name string
		from string
		want string
	}{
		{"a leading double slash", "//evil.com", "/r/everything"},
		{"an absolute URL to another host", "https://evil.com", "/r/everything"},
		{"a leading backslash", `/\evil.com`, "/r/everything"},
		{"a slash then a backslash", `/\/evil.com`, "/r/everything"},
		{"a plain path", "/r/everything", "/r/everything"},
		{"a path with a query", "/board?said=ok", "/board?said=ok"},
		{"empty", "", "/r/everything"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := backTolerant(c.from); got != c.want {
				t.Errorf("backTolerant(%q) = %q, want %q", c.from, got, c.want)
			}
		})
	}
}
