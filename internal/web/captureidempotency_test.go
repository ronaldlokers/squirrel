package web

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestARepeatedCaptureKeyIsANoOp(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)

	first := post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}, "key": {"retry-1"}})
	require.Equal(t, "1", first.Header().Get("X-Kept"))

	second := post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}, "key": {"retry-1"}})
	require.Equal(t, "1", second.Header().Get("X-Kept"))

	require.Len(t, f.items, 1,
		"a repeated capture key wrote a second row instead of returning the first")
}

func TestTwoDifferentCaptureKeysAreTwoRows(t *testing.T) {
	f := &fakeStore{}
	m := mounted(t, f)

	post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}, "key": {"retry-1"}})
	post(t, m, "/capture", url.Values{"text": {"ask the garage about the rattle"}, "key": {"retry-2"}})

	require.Len(t, f.items, 2,
		"two distinct captures were folded into one because their keys were not compared")
}
