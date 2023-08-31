// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"errors"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/project"
	"github.com/siderolabs/kres/internal/project/common"
	"github.com/siderolabs/kres/internal/project/meta"
)

// builder keeps state of project contents being built.
type builder struct {
	proj *project.Contents
	meta *meta.Options

	rootPath string

	commonInputs []dag.Node

	lintInputs []dag.Node
	lintTarget dag.Node

	targets []dag.Node
}

type (
	detectFunc func() (bool, error)
	buildFunc  func() error
)

func newBuilder(meta *meta.Options) *builder {
	return &builder{
		proj:     &project.Contents{},
		meta:     meta,
		rootPath: ".",
	}
}

func (builder *builder) build() error {
	buildTarget := common.NewBuild(builder.meta)
	builder.commonInputs = append(builder.commonInputs, buildTarget, common.NewDocker(builder.meta))
	builder.lintTarget = common.NewLint(builder.meta)

	mandatoryReached := false

	for _, projectType := range []struct {
		detect    detectFunc
		build     buildFunc
		mandatory bool
	}{
		{
			detect: builder.DetectGit,
			build:  builder.BuildGit,
		},
		{
			detect: builder.DetectCI,
			build:  builder.BuildCI,
		},
		{
			detect:    builder.DetectJS,
			build:     builder.BuildJS,
			mandatory: true,
		},
		{
			detect:    builder.DetectGolang,
			build:     builder.BuildGolang,
			mandatory: true,
		},
		{
			detect: builder.DetectMarkdown,
			build:  builder.BuildMarkdown,
		},
		{ // custom should be the last in the list, so that step could be hooked up to the build
			detect: builder.DetectCustom,
			build:  builder.BuildCustom,
		},
	} {
		ok, err := projectType.detect()
		if err != nil {
			return err
		}

		if !ok {
			continue
		}

		err = projectType.build()
		if err != nil {
			return err
		}

		if projectType.mandatory {
			mandatoryReached = true
		}
	}

	if !mandatoryReached {
		return errors.New("no Go or JS files were found")
	}

	if len(builder.lintInputs) > 0 {
		builder.lintTarget.AddInput(builder.lintInputs...)

		builder.targets = append(builder.targets, builder.lintTarget)
	}

	all := common.NewAll(builder.meta)
	all.AddInput(builder.targets...)

	release := common.NewRelease(builder.meta)
	rekres := common.NewReKres(builder.meta)
	makeHelp := common.NewMakeHelp(builder.meta)
	conformance := common.NewConformance(builder.meta)
	slackNotify := common.NewSlackNotify(builder.meta)

	release.AddInput(builder.targets...)

	builder.proj.AddTarget(builder.targets...)
	builder.proj.AddTarget(rekres, all, makeHelp, release, slackNotify, conformance)

	return nil
}
