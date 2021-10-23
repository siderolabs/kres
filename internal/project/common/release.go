// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"path/filepath"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/output/drone"
	"github.com/talos-systems/kres/internal/output/makefile"
	"github.com/talos-systems/kres/internal/project/meta"
)

// Release provides common releasr target.
type Release struct {
	meta *meta.Options
	dag.BaseNode
}

// NewRelease initializes Release.
func NewRelease(meta *meta.Options) *Release {
	return &Release{
		BaseNode: dag.NewBaseNode("release"),

		meta: meta,
	}
}

// CompileDrone implements drone.Compiler.
func (release *Release) CompileDrone(output *drone.Output) error {
	output.Step(drone.MakeStep("release-notes").
		OnlyOnTag().
		DependsOn(dag.GatherMatchingInputNames(release, drone.HasDroneOutput())...),
	)

	output.Step(drone.CustomStep(release.Name()).
		Image("plugins/github-release").
		PublishArtifacts(
			filepath.Join(release.meta.ArtifactsPath, "RELEASE_NOTES.md"),
			filepath.Join(release.meta.ArtifactsPath, "*"),
		).
		OnlyOnTag().
		DependsOn("release-notes"),
	)

	return nil
}

// CompileMakefile implements makefile.Compiler.
func (release *Release) CompileMakefile(output *makefile.Output) error {
	output.Target("release-notes").
		Script("mkdir -p $(ARTIFACTS)").
		Script("@ARTIFACTS=$(ARTIFACTS) ./hack/release.sh $@ $(ARTIFACTS)/RELEASE_NOTES.md $(TAG)").
		Phony()

	return nil
}
