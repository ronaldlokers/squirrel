package web

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func whatWasLogged(t *testing.T) *bytes.Buffer {
	t.Helper()
	said := &bytes.Buffer{}
	was := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(said, nil)))
	t.Cleanup(func() { slog.SetDefault(was) })
	return said
}

func TestAReadThatFailsIsNotSilent(t *testing.T) {
	for _, one := range []struct {
		what string
		read func(Store) any
		says string
	}{
		{"the ladder's step", func(s Store) any {
			return stepFor(s, Options{}, withWho(httptest.NewRequest("GET", "/", nil), 1, "sub"))
		}, "reading the step in progress"},
		{"an opened strip's step", func(s Store) any {
			return stepForItem(s, withWho(httptest.NewRequest("GET", "/", nil), 1, "sub"), 7)
		}, "reading the step in progress"},
	} {
		t.Run(one.what, func(t *testing.T) {
			said := whatWasLogged(t)
			f := &fakeStore{err: errTest}

			one.read(f)

			require.Contains(t, said.String(), one.says,
				"%s stopped being drawn and nothing anywhere says why — it is "+
					"indistinguishable from having nothing to show", one.what)
			require.Contains(t, said.String(), "connection refused",
				"the line says something failed without saying what")
		})
	}
}

func TestAReadThatFoundNothingIsSilent(t *testing.T) {
	said := whatWasLogged(t)
	f := &fakeStore{}
	r := withWho(httptest.NewRequest("GET", "/", nil), 1, "sub")

	require.Nil(t, stepFor(f, Options{}, r))
	require.Nil(t, stepForItem(f, r, 7))

	require.Empty(t, said.String(),
		"nothing to show is not a failure and must not be logged as one")
}
