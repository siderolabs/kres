// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"regexp"

	git "github.com/go-git/go-git/v5"

	"github.com/talos-systems/kres/internal/project/common"
)

// DetectGit detects if current directory is git repository.
func (builder *builder) DetectGit() (bool, error) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		// not a git repo, ignore
		return false, nil
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return true, err
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			for _, remoteURL := range remote.Config().URLs {
				matches := regexp.MustCompile(`github\.com[:/]([^/:]+)/([^/]+)\.git$`).FindStringSubmatch(remoteURL)

				if len(matches) == 3 {
					builder.meta.GitHubOrganization = matches[1]
					builder.meta.GitHubRepository = matches[2]

					break
				}
			}
		}
	}

	return true, nil
}

// BuildGit builds steps for Git repository.
func (builder *builder) BuildGit() error {
	builder.commonInputs = append(builder.commonInputs, common.NewRepository(builder.meta))

	return nil
}
