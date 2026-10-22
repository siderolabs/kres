// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"github.com/siderolabs/kres/internal/output/makefile"
)

// gitCommonDirMountVariable is the Makefile variable with the extra `docker run` mount which makes the git metadata
// of the checkout available to a container working on the source tree.
//
// In a git worktree, `.git` is a file pointing into the main checkout's `.git` directory, so mounting the source tree
// alone leaves the container without the repository: the variable then mounts that directory read-only at the same
// path. In a regular checkout the directory is under the source tree already, and the variable is empty.
const gitCommonDirMountVariable = "GIT_COMMON_DIR_MOUNT"

// gitCommonDirMountRef is the Makefile reference to the variable, to be inserted into `docker run` arguments.
const gitCommonDirMountRef = "$(" + gitCommonDirMountVariable + ")"

// declareGitCommonDirMount declares the variable in the common variable group (no-op if already declared).
func declareGitCommonDirMount(output *makefile.Output) {
	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.SimpleVariable(
			gitCommonDirMountVariable,
			// no unbalanced parentheses here: make would take them for the end of $(shell ...)
			`$(shell git_dir=$$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null); `+
				`if [ -n "$$git_dir" ] && [ "$${git_dir#$(PWD)/}" = "$$git_dir" ]; `+
				`then echo "-v $$git_dir:$$git_dir:ro"; fi)`,
		))
}
