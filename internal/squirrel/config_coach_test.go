package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// No key is the default and a supported state: the picker and the ladder
// answer, and nothing about the product stops working.
func TestLoadConfigLeavesTheCoachOff(t *testing.T) {
	config, err := squirrel.LoadConfig(minimalEnv(nil))
	require.NoError(t, err)
	require.False(t, config.Coach.Enabled())
	require.Empty(t, config.Coach.APIKey)
	// The models and the ceiling still have their defaults, so turning the
	// coach on is one variable rather than four.
	require.Equal(t, "gpt-5.6-luna", config.Coach.Fast)
	require.Equal(t, "gpt-5.6-terra", config.Coach.Deep)
	require.Equal(t, "https://api.openai.com/v1", config.Coach.BaseURL)
	require.Equal(t, int64(10_000_000), config.Coach.BudgetMicros)
}

func TestLoadConfigTurnsTheCoachOnWithAKeyAlone(t *testing.T) {
	config, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"OPENAI_API_KEY": "sk-test",
	}))
	require.NoError(t, err)
	require.True(t, config.Coach.Enabled())
}

// The ceiling is set in whole euros because that is the sentence it exists to
// hold; it is stored in micro-euros because that is what a call costs.
func TestLoadConfigReadsTheBudgetInWholeEuros(t *testing.T) {
	config, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"COACH_BUDGET_EUR": "4",
	}))
	require.NoError(t, err)
	require.Equal(t, int64(4_000_000), config.Coach.BudgetMicros)
}

// Zero disables the in-process ceiling. It has to be reachable, or the only
// way to opt out would be to edit the code.
func TestLoadConfigAllowsNoCeiling(t *testing.T) {
	config, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"COACH_BUDGET_EUR": "0",
	}))
	require.NoError(t, err)
	require.Equal(t, int64(0), config.Coach.BudgetMicros)
}

// Pointing at a gateway is a deployment change rather than a code change, which
// is what makes "gateway-shaped internally" true rather than aspirational.
func TestLoadConfigPointsAtAnotherProvider(t *testing.T) {
	config, err := squirrel.LoadConfig(minimalEnv(map[string]string{
		"OPENAI_API_KEY":   "sk-test",
		"COACH_BASE_URL":   "http://gateway.internal/v1",
		"COACH_MODEL_FAST": "something-else",
	}))
	require.NoError(t, err)
	require.Equal(t, "http://gateway.internal/v1", config.Coach.BaseURL)
	require.Equal(t, "something-else", config.Coach.Fast)
}

// A key with the models blanked out is not half a coach; it is no coach. The
// alternative is calling a model named "".
func TestCoachNeedsBothModels(t *testing.T) {
	require.False(t, squirrel.CoachConfig{APIKey: "sk", Fast: "luna"}.Enabled())
	require.False(t, squirrel.CoachConfig{Fast: "luna", Deep: "terra"}.Enabled())
	require.True(t, squirrel.CoachConfig{APIKey: "sk", Fast: "luna", Deep: "terra"}.Enabled())
}
