// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
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

	{
		list, err := listFilesWithSuffix(builder.rootPath, ".md")
		if err != nil {
			return false, err
		}

		for _, item := range list {
			builder.meta.SourceFiles = append(builder.meta.SourceFiles, item)
			builder.meta.MarkdownSourceFiles = append(builder.meta.MarkdownSourceFiles, item)
		}
	}

	return len(builder.meta.MarkdownDirectories)+len(builder.meta.MarkdownSourceFiles) > 0, nil
}

// BuildMarkdown builds project structure for Markdown.
func (builder *builder) BuildMarkdown() error {
	// linters
	linter := markdown.NewLint(builder.meta)

	builder.lintInputs = append(builder.lintInputs, linter)

	return nil
}
