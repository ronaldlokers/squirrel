package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Everything about you this product can be asked to change is in one place,
// and that place is where it says who you are.
//
// It was three transient chips before 31 August 2026: a floating button for
// notifications that hid itself once answered either way, "what do you know
// about me" beside one model-written reply, and the mood history behind a chip
// that appeared for one turn after a check-in. None of the three had a state
// you could see or come back to.
func TestSettingsHoldsWhatCanBeChangedAboutYou(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	body := withPush(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, `class="youare"`, "there is no settings panel")
	require.Contains(t, body, "Ronald Lokers", "the panel does not say who you are")
	require.Contains(t, body, "Notifications")
	require.Contains(t, body, "What Squirrel knows about you")
	require.Contains(t, body, `action="/auth/out"`)
}

// It is a disclosure and not a page: settings is state rather than a
// conversation, and this product has no third thing for it to be.
func TestSettingsOpensWhereItLives(t *testing.T) {
	body := withPush(t, &fakeStore{}).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, "<details class=\"youare\">")
	require.NotContains(t, body, `href="/settings"`, "settings became somewhere to go")
}

// The panel says which way notifications are set, from the record. Only the
// browser knows whether the permission was refused, and the script says so —
// but "would anything be sent at all" is the server's to answer.
func TestTheRecordSaysWhetherNotificationsAreOn(t *testing.T) {
	on := &fakeStore{notifying: true}
	require.Contains(t, withPush(t, on).call(t, "GET", "/r/everything", nil).Body.String(),
		`data-state="on"`)

	off := &fakeStore{}
	require.Contains(t, withPush(t, off).call(t, "GET", "/r/everything", nil).Body.String(),
		`data-state="off"`)
}

// A state that cannot be read is not a state to guess at.
func TestAnUnreadableNotificationStateSaysSo(t *testing.T) {
	f := &fakeStore{notifyErr: errTest}
	body := withPush(t, f).call(t, "GET", "/r/everything", nil).Body.String()

	require.Contains(t, body, `data-state="unknown"`)
	require.Contains(t, body, "I cannot tell just now")
}

// Turning them off retires every browser, and says nothing back: a setting is
// not something said, so it belongs in no room and draws no turn.
func TestTurningNotificationsOffRetiresThem(t *testing.T) {
	f := &fakeStore{notifying: true}
	res := withPush(t, f).call(t, "POST", "/push/forget", strings.NewReader(""))

	require.Equal(t, 204, res.Code)
	require.True(t, f.stopped, "nothing was retired")
	require.Empty(t, f.appended, "a setting was written into the conversation")
}
