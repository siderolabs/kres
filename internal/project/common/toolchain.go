// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import "github.com/siderolabs/kres/internal/output/dockerfile"

// ToolchainBuilder is implemented by nodes which wish to inject into the toolchain build.
type ToolchainBuilder interface {
	ToolchainBuild(*dockerfile.Stage) error
}

// SourceTreeBuilder is implemented by nodes which inject content into the source tree of the
// golang base stage, before any Go package loading happens there.
type SourceTreeBuilder interface {
	SourceTreeBuild(*dockerfile.Stage) error
}
