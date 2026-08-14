package squirrel_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestNameIsSquirrel(t *testing.T) {
	require.Equal(t, "squirrel", squirrel.Name)
}
