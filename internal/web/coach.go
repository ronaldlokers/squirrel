package web

import "strings"

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
	if !strings.HasPrefix(from, "/") {
		return "/"
	}
	if strings.HasPrefix(strings.ReplaceAll(from, `\`, "/"), "//") {
		return "/"
	}
	return from
}
