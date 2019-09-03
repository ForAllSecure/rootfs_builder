package rootfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPull(t *testing.T) {
	pullable, err := NewPullableImage("../config.json")
	require.NoError(t, err)
	_, err = pullable.Pull()
	require.NoError(t, err)
}
