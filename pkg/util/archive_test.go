package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSevenZipArchiveArgsEnablesHeaderEncryption(t *testing.T) {
	args := sevenZipArchiveArgs("backup.7z", ".", "correct horse battery staple")

	assert.Contains(t, args, "-t7z")
	assert.Contains(t, args, "-mhe=on")
	assert.Contains(t, args, "-pcorrect horse battery staple")
	assert.Equal(t, []string{"--", "backup.7z", "."}, args[len(args)-3:])
}

func TestSevenZipWithPasswordRejectsEmptyPassword(t *testing.T) {
	err := SevenZipWithPassword(t.TempDir(), filepath.Join(t.TempDir(), "backup.7z"), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestSevenZipWithPasswordReportsMissingSource(t *testing.T) {
	err := SevenZipWithPassword(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "backup.7z"), "strong-password")

	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFindSevenZipExecutableUsesConfiguredPath(t *testing.T) {
	tempDir := t.TempDir()
	name := "custom-7z"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable := filepath.Join(tempDir, name)
	require.NoError(t, os.WriteFile(executable, []byte{}, 0o755))
	t.Setenv("SEVENZIP_BINARY", executable)

	actual, err := findSevenZipExecutable()

	require.NoError(t, err)
	assert.Equal(t, executable, actual)
}

func TestFindSevenZipBesideExecutable(t *testing.T) {
	tempDir := t.TempDir()
	name := "7zz"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	want := filepath.Join(tempDir, name)
	require.NoError(t, os.WriteFile(want, []byte{}, 0o755))

	actual, ok := findSevenZipBesideExecutable(filepath.Join(tempDir, "fast-note-sync-service"))

	require.True(t, ok)
	assert.Equal(t, want, actual)
}
