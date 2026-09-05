package web

import "testing"

func TestBackTolerantRejectsEveryWayOutOfTheHost(t *testing.T) {
	cases := []struct {
		name string
		from string
		want string
	}{
		{"a leading double slash", "//evil.com", "/"},
		{"an absolute URL to another host", "https://evil.com", "/"},
		{"a leading backslash", `/\evil.com`, "/"},
		{"a slash then a backslash", `/\/evil.com`, "/"},
		{"a plain path", "/?open=1", "/?open=1"},
		{"a path with a query", "/board?said=ok", "/board?said=ok"},
		{"empty", "", "/"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := backTolerant(c.from); got != c.want {
				t.Errorf("backTolerant(%q) = %q, want %q", c.from, got, c.want)
			}
		})
	}
}
