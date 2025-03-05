package bldr

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

type RandomBuilder struct {
	t *testing.T
}

func Random(t *testing.T) RandomBuilder {
	return RandomBuilder{t: t}
}

func (b RandomBuilder) String() string {
	bytes := make([]byte, 8)
	_, err := rand.Read(bytes)
	require.NoError(b.t, err)

	return hex.EncodeToString(bytes)
}
