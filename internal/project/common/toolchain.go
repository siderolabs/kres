// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import "github.com/siderolabs/kres/internal/output/dockerfile"

// ToolchainBuilder is implemented by nodes which wish to inject into the toolchain build.
type ToolchainBuilder interface {
	ToolchainBuild(*dockerfile.Stage) error
}
