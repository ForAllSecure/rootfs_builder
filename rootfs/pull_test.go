package rootfs

import (
	"strings"
	"testing"
	"time"

	"github.com/ForAllSecure/rootfs_builder/rootfs"
	"github.com/stretchr/testify/require"
)

func TestPull(t *testing.T) {
	pullable, err := rootfs.NewPullableImage("../test/alpine.json")
	require.NoError(t, err)
	_, err = pullable.Pull()
	require.NoError(t, err)
}

func TestPullTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	pullable, err := rootfs.NewPullableImage("../test/timeout.json")
	require.NoError(t, err)

	start := time.Now()
	_, err = pullable.Pull()
	require.Less(t, int64(time.Since(start)/time.Second), int64(60))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "i/o timeout"))
}
