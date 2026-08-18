package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The prefix rule is the whole defence against a command eating a thought.
// Every one of these is something a person might actually type at a bot whose
// entire job is to remember what they said.
func TestBareWordsAreStillCaptures(t *testing.T) {
	for _, raw := range []string{
		"find my keys",
		"find the receipt for the boiler",
		"notes to self about the boiler",
		"notes from the meeting",
		"help me remember to call the dentist",
		"help",
		"chore the bins tomorrow",
		"chores are the worst",
	} {
		require.Equal(t, squirrel.IntentCapture, squirrel.Match(raw).Kind,
			"a bare word must never become a command: %q", raw)
	}
}

func TestCommandsNeedThePrefix(t *testing.T) {
	in := squirrel.Match("!find boiler")
	require.Equal(t, squirrel.IntentCommand, in.Kind)
	require.Equal(t, "find", in.Command)
	require.Equal(t, "boiler", in.Arg)
}

func TestACommandWithNoArgumentHasAnEmptyArg(t *testing.T) {
	in := squirrel.Match("!notes")
	require.Equal(t, squirrel.IntentCommand, in.Kind)
	require.Equal(t, "notes", in.Command)
	require.Empty(t, in.Arg)
}

// The argument is matched against text the user wrote and, on the promotion
// path, becomes a chore's name. Its case is theirs.
func TestCommandArgKeepsItsCase(t *testing.T) {
	require.Equal(t, "Boiler Serial", squirrel.Match("!find Boiler Serial").Arg)
}

func TestCommandNameIsCaseInsensitive(t *testing.T) {
	require.Equal(t, "find", squirrel.Match("!FIND boiler").Command)
	require.Equal(t, "find", squirrel.Match("!Find boiler").Command)
}

// A typo answered with 👀 would be filed as a note, silently, and the
// correction with it.
func TestAnUnknownCommandIsNotACapture(t *testing.T) {
	in := squirrel.Match("!fnid boiler")
	require.Equal(t, squirrel.IntentCommand, in.Kind)
	require.Equal(t, "fnid", in.Command)
}

// `.` is how you capture a thought shaped like a command.
//
// This does not pin an ordering, and it would be easy to read it as though it
// did: ".!find" does not start with "!", so the two prefix checks can never
// both match and swapping them changes nothing. What it pins is that the
// escaped text arrives verbatim, with the "." gone and the "!" kept.
func TestTheEscapeHatchBeatsThePrefix(t *testing.T) {
	in := squirrel.Match(".!find boiler")
	require.Equal(t, squirrel.IntentCapture, in.Kind)
	require.Equal(t, "!find boiler", in.Text)
}

func TestABareBangIsACapture(t *testing.T) {
	require.Equal(t, squirrel.IntentCapture, squirrel.Match("!").Kind)
	require.Equal(t, squirrel.IntentCapture, squirrel.Match("!!!").Kind)
	require.Equal(t, squirrel.IntentCapture, squirrel.Match("!   ").Kind)
}

// A bare "!" captures the raw text, not a stripped copy: the raw text is the
// record, exactly as it is for every other capture.
func TestABareBangCapturesVerbatim(t *testing.T) {
	require.Equal(t, "!!!", squirrel.Match("!!!").Text)
}

func TestCommandExtraSpacingIsTolerated(t *testing.T) {
	in := squirrel.Match("!find    boiler serial")
	require.Equal(t, "find", in.Command)
	require.Equal(t, "boiler serial", in.Arg,
		"the gap between command and argument is typing, not meaning")
}

// Everything phases 1 to 4 recognise keeps working. `?` in particular is
// muscle memory and phase 4 leaned on it as the escape from the nudge budget.
func TestExistingIntentsAreUnaffected(t *testing.T) {
	require.Equal(t, squirrel.IntentQuery, squirrel.Match("?").Kind)
	require.Equal(t, squirrel.IntentComplete, squirrel.Match("done").Kind)
	require.Equal(t, squirrel.IntentComplete, squirrel.Match("done 2").Kind)
	require.Equal(t, squirrel.IntentComplete, squirrel.Match("2").Kind)
	require.Equal(t, squirrel.IntentStop, squirrel.Match("stop 1").Kind)
	require.Equal(t, squirrel.IntentDrop, squirrel.Match("nvm").Kind)
	require.Equal(t, squirrel.IntentDefine, squirrel.Match("every 2 weeks vacuum").Kind)
}
