// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package auto provides automatic detector of project type, reflections.
package auto

import (
	"os"
	"path/filepath"

	"github.com/talos-systems/kres/internal/dag"
	"github.com/talos-systems/kres/internal/project"
	"github.com/talos-systems/kres/internal/project/common"
	"github.com/talos-systems/kres/internal/project/meta"
)

type (
	detector func(string, *meta.Options) (bool, error)
	builder  func(*meta.Options, []dag.Node) ([]dag.Node, error)
)

// Build the project type and structure based on project type.
func Build(meta *meta.Options) (*project.Contents, error) {
	proj := &project.Contents{}

	inputs := []dag.Node{common.NewBuild(meta), common.NewDocker(meta)}
	outputs := []dag.Node{}

	for _, projectType := range []struct {
		detect detector
		build  builder
	}{
		{
			detect: DetectGolang,
			build:  BuildGolang,
		},
	} {
		ok, err := projectType.detect(".", meta)
		if err != nil {
			return nil, err
		}

		if !ok {
			continue
		}

		newOutputs, err := projectType.build(meta, inputs)
		if err != nil {
			return nil, err
		}

		outputs = append(outputs, newOutputs...)
	}

	all := common.NewAll(meta)
	all.AddInput(outputs...)

	rekres := common.NewReKres(meta)

	makeHelp := common.NewMakeHelp(meta)

	proj.AddTarget(outputs...)
	proj.AddTarget(rekres, all, makeHelp)

	return proj, nil
}

func directoryExists(rootPath, name string) (bool, error) {
	path := filepath.Join(rootPath, name)

	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return st.IsDir(), nil
}
