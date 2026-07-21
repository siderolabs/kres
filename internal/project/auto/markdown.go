// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/project/markdown"
)

// DetectMarkdown checks if project at rootPath contains Markdown files.
func (builder *builder) DetectMarkdown() (bool, error) {
	for _, srcDir := range []string{"docs"} {
		exists, err := directoryExists(builder.rootPath, srcDir)
		if err != nil {
			return false, err
		}

		if exists {
			builder.meta.Directories = append(builder.meta.Directories, srcDir)
			builder.meta.MarkdownDirectories = append(builder.meta.MarkdownDirectories, srcDir)
		}
	}

	list, err := listFilesWithSuffix(builder.rootPath, ".md")
	if err != nil {
		return false, err
	}

	tracked, inRepo := trackedFiles(builder.rootPath)

	for _, item := range list {
		// Inside a git repository, only wire in tracked markdown so that a
		// developer's personal, untracked file (for example a git-ignored
		// CLAUDE.local.md) is neither copied into the generated Dockerfile nor
		// picked up by the markdown linter. Outside a repository there is no
		// basis to filter, so keep every file.
		if inRepo {
			if _, ok := tracked[item]; !ok {
				continue
			}
		}

		builder.meta.SourceFiles = append(builder.meta.SourceFiles, item)
		builder.meta.MarkdownSourceFiles = append(builder.meta.MarkdownSourceFiles, item)
	}

	return len(builder.meta.MarkdownDirectories)+len(builder.meta.MarkdownSourceFiles) > 0, nil
}

// BuildMarkdown builds project structure for Markdown.
func (builder *builder) BuildMarkdown() error {
	if builder.meta.ContainerImageFrontend != config.ContainerImageFrontendDockerfile {
		return nil
	}

	// linters
	linter := markdown.NewLint(builder.meta)

	builder.lintInputs = append(builder.lintInputs, linter)

	return nil
}
