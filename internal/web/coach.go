package web

import (
	"net/url"
	"strings"
)

type Exchange struct {
	Said    string
	Replied string
}

type Answer struct {
	Text    string
	Did     []string
	Propose *Proposal
	Open    string
}

type Proposal struct {
	Do    string
	Said  string
	Text  string
	At    string
	Every string
	RefID int64
}

func coachAvailable(opts Options) bool { return opts.Ask != nil }

func remember(opts Options, personID int64, room, said, replied string) {
	if opts.Remember != nil {
		opts.Remember(personID, room, said, replied)
	}
}

func withDid(a Answer) string {
	if len(a.Did) == 0 {
		return a.Text
	}
	return a.Text + " (" + strings.Join(a.Did, "; ") + ")"
}

func backTolerant(from string) string {
	u, err := url.Parse(from)
	if err != nil || u.Scheme != "" || u.Opaque != "" || u.Host != "" ||
		!strings.HasPrefix(u.Path, "/") || strings.Contains(from, `\`) {
		return "/"
	}
	return u.String()
}
