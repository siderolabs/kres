// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/project/auto"
)

func writeMarkdown(t *testing.T, dir, name string) {
	t.Helper()

	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("# "+name+"\n"), 0o600))
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
}

// TestDetectMarkdownOnlyTracked verifies that markdown auto-detection wires in
// tracked files (committed or staged) but skips a developer's personal,
// untracked file such as a git-ignored CLAUDE.local.md.
func TestDetectMarkdownOnlyTracked(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	root := t.TempDir()

	initGitRepo(t, root)

	// Committed team files.
	for _, name := range []string{"README.md", "AGENTS.md", "CLAUDE.md"} {
		writeMarkdown(t, root, name)
	}

	runGit(t, root, "add", "README.md", "AGENTS.md", "CLAUDE.md")
	runGit(t, root, "commit", "-m", "init")

	// A staged-but-not-committed file is tracked via the index.
	writeMarkdown(t, root, "GEMINI.md")
	runGit(t, root, "add", "GEMINI.md")

	// A personal, untracked file.
	writeMarkdown(t, root, "CLAUDE.local.md")

	files, detected, err := auto.DetectMarkdownSourceFiles(root)
	require.NoError(t, err)
	assert.True(t, detected)

	assert.ElementsMatch(t, []string{"README.md", "AGENTS.md", "CLAUDE.md", "GEMINI.md"}, files)
	assert.NotContains(t, files, "CLAUDE.local.md")
}

// TestDetectMarkdownInLinkedWorktree verifies detection uses the per-worktree
// index in a linked worktree, where .git is a file rather than a directory.
func TestDetectMarkdownInLinkedWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	root := t.TempDir()

	initGitRepo(t, root)

	writeMarkdown(t, root, "AGENTS.md")
	runGit(t, root, "add", "AGENTS.md")
	runGit(t, root, "commit", "-m", "init")
	runGit(t, root, "branch", "-M", "main")

	worktree := filepath.Join(t.TempDir(), "worktree")
	runGit(t, root, "worktree", "add", "-b", "feature", worktree)

	// A personal, untracked file in the worktree.
	writeMarkdown(t, worktree, "CLAUDE.local.md")

	files, detected, err := auto.DetectMarkdownSourceFiles(worktree)
	require.NoError(t, err)
	assert.True(t, detected)

	assert.ElementsMatch(t, []string{"AGENTS.md"}, files)
	assert.NotContains(t, files, "CLAUDE.local.md")
}

// TestDetectMarkdownOutsideRepositoryKeepsAll verifies that outside a git
// repository every markdown file is kept, preserving the previous behavior.
func TestDetectMarkdownOutsideRepositoryKeepsAll(t *testing.T) {
	root := t.TempDir()

	for _, name := range []string{"README.md", "NOTES.md"} {
		writeMarkdown(t, root, name)
	}

	files, detected, err := auto.DetectMarkdownSourceFiles(root)
	require.NoError(t, err)
	assert.True(t, detected)

	assert.ElementsMatch(t, []string{"README.md", "NOTES.md"}, files)
}
