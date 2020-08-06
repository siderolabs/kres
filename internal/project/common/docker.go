// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Docker provides build infrastructure via docker buildx.
type Docker struct {
	dag.BaseNode

	meta *meta.Options
}

// NewDocker initializes Docker.
func NewDocker(meta *meta.Options) *Docker {
	meta.BuildArgs = append(meta.BuildArgs, "USERNAME")

	return &Docker{
		meta: meta,
	}
}

// CompileMakefile implements makefile.Compiler.
func (docker *Docker) CompileMakefile(output *makefile.Output) error {
	buildArgs := makefile.RecursiveVariable("COMMON_ARGS", "--file=Dockerfile").
		Push("--progress=$(PROGRESS)").
		Push("--platform=$(PLATFORM)").
		Push("--push=$(PUSH)")

	for _, arg := range docker.meta.BuildArgs {
		buildArgs.Push(fmt.Sprintf("--build-arg=%s=$(%s)", arg, arg))
	}

	output.VariableGroup(makefile.VariableGroupCommon).
		Variable(makefile.OverridableVariable("REGISTRY", "docker.io")).
		Variable(makefile.OverridableVariable("USERNAME", "autonomy")).
		Variable(makefile.OverridableVariable("REGISTRY_AND_USERNAME", "$(REGISTRY)/$(USERNAME)"))

	output.VariableGroup(makefile.VariableGroupDocker).
		Variable(makefile.SimpleVariable("BUILD", "docker buildx build")).
		Variable(makefile.OverridableVariable("PLATFORM", "linux/amd64")).
		Variable(makefile.OverridableVariable("PROGRESS", "auto")).
		Variable(makefile.OverridableVariable("PUSH", "false")).
		Variable(makefile.OverridableVariable("CI_ARGS", "")).
		Variable(buildArgs)

	output.Target("target-%").
		Description("Builds the specified target defined in the Dockerfile. The build result will only remain in the build cache.").
		Script(`@$(BUILD) --target=$* $(COMMON_ARGS) $(TARGET_ARGS) $(CI_ARGS) .`)

	output.Target("local-%").
		Description("Builds the specified target defined in the Dockerfile using the local output type. The build result will be output to the specified local destination.").
		Script(`@$(MAKE) target-$* TARGET_ARGS="--output=type=local,dest=$(DEST) $(TARGET_ARGS)"`)

	return nil
}
