// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/project/auto"
	"github.com/siderolabs/kres/internal/project/meta"
)

func TestDetectGitWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	root := filepath.Join(t.TempDir(), "repo")
	worktree := filepath.Join(t.TempDir(), "worktree")

	runGit(t, "", "init", root)
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "config", "commit.gpgsign", "false")
	runGit(t, root, "commit", "--allow-empty", "-m", "init")
	runGit(t, root, "branch", "-M", "main")
	runGit(t, root, "remote", "add", "origin", "git@github.com:siderolabs/example.git")
	runGit(t, root, "config", "branch.main.remote", "origin")
	runGit(t, root, "config", "branch.main.merge", "refs/heads/main")
	runGit(t, root, "worktree", "add", "-b", "feature", worktree)

	t.Chdir(worktree)

	options := &meta.Options{
		CompileGithubWorkflowsOnly: true,
	}

	_, err := auto.Build(options)
	require.NoError(t, err)

	assert.Equal(t, "main", options.MainBranch)
	assert.Equal(t, "feature", options.CurrentBranch)
	assert.Equal(t, "siderolabs", options.GitHubOrganization)
	assert.Equal(t, "example", options.GitHubRepository)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
