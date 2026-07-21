// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
)

func listFilesWithSuffix(path, suffix string) ([]string, error) {
	contents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []string

	for _, item := range contents {
		if !item.IsDir() && strings.HasSuffix(item.Name(), suffix) {
			result = append(result, item.Name())
		}
	}

	return result, nil
}

// trackedFiles returns the set of file names tracked in the git index of the
// repository at rootPath, so kres wires only committed or staged files into
// generated output and never a developer's personal, untracked file (for
// example a git-ignored CLAUDE.local.md). The map is keyed by the index entry
// name (repository-root-relative, forward-slash separated).
//
// inRepo is false when rootPath is not a git repository, or when the index
// cannot be read for any reason: an unopenable repository, or an index format
// the vendored go-git cannot decode (for example a split index or a SHA-256
// repository). In those cases callers must not filter. This both mirrors
// DetectGit's tolerance of unreadable repositories and guarantees that
// generation degrades to the previous "wire in every markdown file" behavior
// instead of failing on an otherwise valid repository. EnableDotGitCommonDir
// mirrors DetectGit so linked worktrees, where .git is a file, use the correct
// per-worktree index.
func trackedFiles(rootPath string) (tracked map[string]struct{}, inRepo bool) {
	repo, err := git.PlainOpenWithOptions(rootPath, &git.PlainOpenOptions{
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, false //nolint:nilerr // treat any unopenable repo as "not a repo": do not filter
	}

	index, err := repo.Storer.Index()
	if err != nil {
		return nil, false //nolint:nilerr // unreadable/unsupported index: degrade to no filtering rather than fail
	}

	tracked = make(map[string]struct{}, len(index.Entries))

	for _, entry := range index.Entries {
		tracked[entry.Name] = struct{}{}
	}

	return tracked, true
}

func directoryExists(rootPath, name string) (bool, error) {
	path := filepath.Join(rootPath, name)

	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return st.IsDir(), nil
}
