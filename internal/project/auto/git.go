// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"errors"
	"fmt"
	"regexp"

	git "github.com/go-git/go-git/v5"

	"github.com/siderolabs/kres/internal/project/common"
)

// DetectGit detects if current directory is git repository.
func (builder *builder) DetectGit() (bool, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		// not a git repo, ignore
		return false, nil //nolint:nilerr
	}

	c, err := repo.Config()
	if err != nil {
		return true, fmt.Errorf("failed to get repository configuration: %w", err)
	}

	rawConfig := c.Raw

	const (
		main              = "main"
		branchSectionName = "branch"
	)

	if !rawConfig.HasSection(branchSectionName) {
		return true, fmt.Errorf("repository configuration section %q not found", branchSectionName)
	}

	branchSection := rawConfig.Section(branchSectionName)

	for _, b := range branchSection.Subsections {
		if b.Name == main {
			builder.meta.MainBranch = main

			break
		}

		remote := b.Option("remote")
		if remote == git.DefaultRemoteName {
			builder.meta.MainBranch = b.Name
		}
	}

	if builder.meta.MainBranch == "" {
		builder.meta.MainBranch = main
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return true, err
	}

	var upstreamRemote *git.Remote

	for _, remote := range remotes {
		if remote.Config().Name == "upstream" {
			upstreamRemote = remote

			break
		}
	}

	if upstreamRemote == nil {
		for _, remote := range remotes {
			if remote.Config().Name == "origin" {
				upstreamRemote = remote

				break
			}
		}
	}

	if upstreamRemote == nil {
		return true, errors.New("neither 'origin' or 'upstream' remote found")
	}

	remoteURLregexp := `((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,})[:/]+([^/:]+)/([^/]+)\.git$`
	for _, remoteURL := range upstreamRemote.Config().URLs {
		matches := regexp.MustCompile(remoteURLregexp).FindStringSubmatch(remoteURL)
		if len(matches) == 4 {
			if matches[1] != "github.com" {
				return false, nil //nolint:nilerr
			}

			builder.meta.GitHubOrganization = matches[2]
			builder.meta.GitHubRepository = matches[3]

			return true, nil
		}
	}

	return true, fmt.Errorf("failed to parse remote URL: %s", upstreamRemote)
}

// BuildGit builds steps for Git repository.
func (builder *builder) BuildGit() error {
	builder.commonInputs = append(
		builder.commonInputs,
		common.NewRepository(builder.meta),
		common.NewCheckDirty(builder.meta),
	)

	return nil
}
