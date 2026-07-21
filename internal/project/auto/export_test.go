// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import "github.com/siderolabs/kres/internal/project/meta"

// DetectMarkdownSourceFiles runs markdown auto-detection at rootPath and returns
// the detected markdown source files along with whether any markdown was
// detected. It is exposed for external tests.
func DetectMarkdownSourceFiles(rootPath string) ([]string, bool, error) {
	options := &meta.Options{}

	builder := newBuilder(options)
	builder.rootPath = rootPath

	detected, err := builder.DetectMarkdown()

	return options.MarkdownSourceFiles, detected, err
}
