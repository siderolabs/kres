// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/project"
	"github.com/talos-systems/kres/internal/project/common"
	"github.com/talos-systems/kres/internal/project/meta"
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
	builder.commonInputs = append(builder.commonInputs, common.NewBuild(builder.meta), common.NewDocker(builder.meta))
	builder.lintTarget = common.NewLint(builder.meta)

	for _, projectType := range []struct {
		detect detectFunc
		build  buildFunc
	}{
		{
			detect: builder.DetectGolang,
			build:  builder.BuildGolang,
		},
		{
			detect: builder.DetectMarkdown,
			build:  builder.BuildMarkdown,
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
	}

	if len(builder.lintInputs) > 0 {
		builder.lintTarget.AddInput(builder.lintInputs...)

		builder.targets = append(builder.targets, builder.lintTarget)
	}

	all := common.NewAll(builder.meta)
	all.AddInput(builder.targets...)

	rekres := common.NewReKres(builder.meta)
	makeHelp := common.NewMakeHelp(builder.meta)

	builder.proj.AddTarget(builder.targets...)
	builder.proj.AddTarget(rekres, all, makeHelp)

	return nil
}
