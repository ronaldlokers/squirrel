//go:build integration

package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Setting something aside, in chat. The number is a line on the last numbered
// surface, the same gesture as `done 3` — no new way to point at a note.

func TestWaitingTakesALineOutOfThePile(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	id := lineItemID(t, store, p, 1)

	reply := triage(t, store, p, "!waiting 1 on the vet")
	require.Contains(t, reply, "ring the vet")
	require.Contains(t, reply, "waiting on the vet")

	require.Equal(t, "waiting", stateOf(t, store, id))
}

// "on" is how the sentence reads and is not part of who you are waiting on.
func TestTheLeadingOnIsNotPartOfTheReason(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	require.Contains(t, triage(t, store, p, "!waiting 1 on the vet"), "waiting on the vet")

	pileOf(t, store, p, "fix the boiler")
	require.Contains(t, triage(t, store, p, "!blocked 1 the part"), "blocked on the part")
}

func TestSomedayNeedsNothingAfterTheNumber(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "learn to solder")
	id := lineItemID(t, store, p, 1)

	require.Contains(t, triage(t, store, p, "!someday 1"), "someday")
	require.Equal(t, "someday", stateOf(t, store, id))
}

// With no number it is a request to see the list. Setting things aside is only
// safe if they are easy to find again.
func TestWaitingWithNoNumberListsWhatIsSetAside(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	pileOf(t, store, p, "ring the vet")
	triage(t, store, p, "!waiting 1 on the vet")
	pileOf(t, store, p, "learn to solder")
	triage(t, store, p, "!someday 1")

	reply := triage(t, store, p, "!waiting")
	require.Contains(t, reply, "ring the vet")
	require.Contains(t, reply, "the vet")
	require.Contains(t, reply, "learn to solder")
	// Grouped, because they are different questions.
	require.Contains(t, reply, "WAITING ON")
	require.Contains(t, reply, "SOMEDAY")
}

func TestNothingSetAsideSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	require.Contains(t, triage(t, store, p, "!someday"), "Nothing set aside")
}

// No count anywhere, in either direction. A number beside stalled work is a
// reproach, and the point of setting it aside was to stop being asked.
func TestTheListNeverSaysHowMany(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	for _, text := range []string{"one", "two", "three"} {
		pileOf(t, store, p, text)
		triage(t, store, p, "!someday 1")
	}

	reply := triage(t, store, p, "!waiting")
	for _, count := range []string{"3", "three things", "3 items"} {
		require.NotContains(t, reply, count)
	}
}

func TestSettingAsideSomethingThatIsNotThereSaysSo(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	require.Contains(t, triage(t, store, p, "!waiting 4 on the vet"), "No line 4")
	require.Contains(t, triage(t, store, p, "!waiting the vet"), "Which one?")
}

// Help lists them, because a way to set something aside that nobody knows
// about is a way nobody uses.
func TestHelpNamesTheThreeWays(t *testing.T) {
	help := squirrel.HelpMessage().Text
	require.Contains(t, help, "!waiting <n> on <who>")
	require.Contains(t, help, "!someday <n>")
}
