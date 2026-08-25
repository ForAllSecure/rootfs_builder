package rootfs

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// Real reproduction of GitHub issue #8: extended attributes on a tar
// entry are silently dropped during extraction. Builds a REAL tar stream
// (Go's own archive/tar writer, not a mock) with a PAX xattr record on a
// regular file, runs it through the actual handleFiles() extraction
// path, and checks the resulting file's xattr via the real Lgetxattr
// syscall -- not asserting on internal state, on the actual filesystem
// object the tool produced.
func TestExtractedFileKeepsItsXattr(t *testing.T) {
	dest := t.TempDir()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	content := []byte("hello world\n")
	hdr := &tar.Header{
		Name: "testfile",
		Mode: 0644,
		// Uid/Gid left at zero; extractFile adds subuid/subgid to these,
		// so setting them to the CURRENT process's own ids below lets
		// the resulting chown succeed without root -- unrelated to the
		// xattr bug this test targets.
		Size:   int64(len(content)),
		Format: tar.FormatPAX,
		PAXRecords: map[string]string{
			"SCHILY.xattr.user.rootfs_builder_test": "present",
		},
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("writing tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("writing tar content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}

	tr := tar.NewReader(&buf)
	if err := handleFiles(tr, dest, os.Getuid(), os.Getgid()); err != nil {
		t.Fatalf("handleFiles: %v", err)
	}

	path := filepath.Join(dest, "testfile")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected extracted file to exist: %v", err)
	}

	value := make([]byte, 64)
	n, err := unix.Lgetxattr(path, "user.rootfs_builder_test", value)
	if err != nil {
		t.Fatalf("BUG CONFIRMED (issue #8): extracted file has no "+
			"user.rootfs_builder_test xattr -- Lgetxattr failed: %v", err)
	}
	if got := string(value[:n]); got != "present" {
		t.Fatalf("xattr value mismatch: got %q, want %q", got, "present")
	}
}
