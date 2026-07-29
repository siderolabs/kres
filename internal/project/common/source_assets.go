// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package common

import (
	"fmt"
	"path"
	"strings"

	"github.com/siderolabs/gen/xslices"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output/dockerfile"
	"github.com/siderolabs/kres/internal/output/dockerfile/step"
	"github.com/siderolabs/kres/internal/output/dockerignore"
	"github.com/siderolabs/kres/internal/output/gitignore"
	"github.com/siderolabs/kres/internal/output/makefile"
	"github.com/siderolabs/kres/internal/output/renovate"
	"github.com/siderolabs/kres/internal/project/meta"
)

// SourceAssetsStage is the name of the docker stage which collects all the source assets.
const SourceAssetsStage = "source-assets"

// SourceAssets provides files materialized into the source tree of the build without being committed
// to the repository, e.g. binaries referenced by go:embed directives.
//
// The files are pulled from pinned container images. The docker build copies them from image stages
// into the source tree of the base stage, so every stage built on top of it sees them. For native
// builds outside docker, a generated make target fetches the same files through the same stages, so
// the two paths cannot diverge. The destinations are ignored in git and excluded from the docker
// build context.
//
// Only Go projects consume the assets, and only one document of this kind is honored per project.
// The generated docker stage names use the "source-assets" prefix, which no other stage should use.
type SourceAssets struct { //nolint:govet
	dag.BaseNode

	meta *meta.Options

	Images []SourceAssetsImage `yaml:"images"`
}

// SourceAssetsImage is a pinned container image providing source assets.
type SourceAssetsImage struct {
	Ref    string             `yaml:"ref"`
	Copies []SourceAssetsCopy `yaml:"copies"`
}

// SourceAssetsCopy is a single path copied out of an image into the source tree.
type SourceAssetsCopy struct {
	Platform    string `yaml:"platform"`
	Source      string `yaml:"source"`
	Destination string `yaml:"destination"`
}

// NewSourceAssets initializes SourceAssets.
func NewSourceAssets(meta *meta.Options) *SourceAssets {
	return &SourceAssets{
		BaseNode: dag.NewBaseNode(SourceAssetsStage),
		meta:     meta,
	}
}

// AfterLoad validates and normalizes the configuration.
func (assets *SourceAssets) AfterLoad() error {
	var destinations []string

	for i, image := range assets.Images {
		if image.Ref == "" {
			return fmt.Errorf("source assets image ref is empty")
		}

		if len(image.Copies) == 0 {
			return fmt.Errorf("source assets image %q declares no copies", image.Ref)
		}

		for j, imageCopy := range image.Copies {
			if !strings.HasPrefix(imageCopy.Source, "/") {
				return fmt.Errorf("source assets copy source %q from image %q must be an absolute path", imageCopy.Source, image.Ref)
			}

			destination := path.Clean(imageCopy.Destination)
			if destination == "." || destination == ".." || path.IsAbs(destination) || strings.HasPrefix(destination, "../") {
				return fmt.Errorf("source assets destination %q must be a relative path inside the source tree", imageCopy.Destination)
			}

			for _, other := range destinations {
				if destination == other || strings.HasPrefix(destination, other+"/") || strings.HasPrefix(other, destination+"/") {
					return fmt.Errorf("source assets destinations %q and %q overlap", destination, other)
				}
			}

			destinations = append(destinations, destination)

			assets.Images[i].Copies[j].Destination = destination
		}
	}

	return nil
}

// destinations returns all destination paths in the declaration order.
func (assets *SourceAssets) destinations() []string {
	var out []string

	for _, image := range assets.Images {
		for _, imageCopy := range image.Copies {
			out = append(out, imageCopy.Destination)
		}
	}

	return out
}

// CompileDockerfile implements dockerfile.Compiler.
//
// It emits one stage per unique image and platform, and a scratch collector stage holding every
// copy at its destination path. The base stage and the native fetch target both consume the
// collector, so their contents are identical by construction.
func (assets *SourceAssets) CompileDockerfile(output *dockerfile.Output) error {
	sourceStages := map[string]string{}

	for _, image := range assets.Images {
		for _, imageCopy := range image.Copies {
			key := image.Ref + "|" + imageCopy.Platform

			if _, ok := sourceStages[key]; !ok {
				stageName := fmt.Sprintf("%s-src-%d", SourceAssetsStage, len(sourceStages))
				sourceStages[key] = stageName

				stage := output.Stage(stageName).Description("source assets from " + image.Ref).From(image.Ref)

				if imageCopy.Platform != "" {
					stage.Platform(imageCopy.Platform)
				}
			}
		}
	}

	collector := output.Stage(SourceAssetsStage).
		Description("collects the source assets").
		From("scratch")

	for _, image := range assets.Images {
		for _, imageCopy := range image.Copies {
			collector.Step(step.Copy(imageCopy.Source, "/"+imageCopy.Destination).From(sourceStages[image.Ref+"|"+imageCopy.Platform]))
		}
	}

	return nil
}

// SourceTreeBuild implements SourceTreeBuilder.
func (assets *SourceAssets) SourceTreeBuild(stage *dockerfile.Stage) error {
	stage.Step(step.Copy("/", "./").From(SourceAssetsStage))

	return nil
}

// CompileMakefile implements makefile.Compiler.
//
// The build platform is pinned to a single constant one on purpose: the collector contents are
// platform-independent, since every image stage pins its own platform explicitly, and a
// multi-platform build would split the local export into per-platform subdirectories.
func (assets *SourceAssets) CompileMakefile(output *makefile.Output) error {
	quoted := xslices.Map(assets.destinations(), func(destination string) string { return "'" + destination + "'" })

	output.Target("fetch-source-assets").
		Description("Fetches the source assets into the source tree, for building natively outside docker.").
		Script("rm -rf -- " + strings.Join(quoted, " ")).
		Script("@$(MAKE) local-" + SourceAssetsStage + " DEST=./ PLATFORM=linux/amd64").
		Phony()

	return nil
}

// SkipAsMakefileDependency implements makefile.SkipAsMakefileDependency.
func (assets *SourceAssets) SkipAsMakefileDependency() {}

// CompileGitignore implements gitignore.Compiler.
func (assets *SourceAssets) CompileGitignore(output *gitignore.Output) error {
	output.IgnorePath(assets.destinations()...)

	return nil
}

// CompileDockerignore implements dockerignore.Compiler.
//
// The destinations are excluded from the docker build context, so locally fetched copies never leak
// into the container build, where the same files come from the image stages.
func (assets *SourceAssets) CompileDockerignore(output *dockerignore.Output) error {
	output.DenyLocalPath(assets.destinations()...)

	return nil
}

// CompileRenovate implements renovate.Compiler.
//
// The image references get a custom manager, so they receive update PRs like any other pinned image.
func (assets *SourceAssets) CompileRenovate(output *renovate.Output) error {
	output.CustomManagers([]renovate.CustomManager{
		{
			CustomType:          "regex",
			ManagerFilePatterns: []string{`/^\.kres\.yaml$/`},
			MatchStrings:        []string{`ref:\s*["']?(?<depName>[^\s"']+):(?<currentValue>[A-Za-z0-9._-]+)["']?\s*$`},
			DataSourceTemplate:  "docker",
			VersioningTemplate:  "docker",
		},
	})

	return nil
}
