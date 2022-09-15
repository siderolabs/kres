// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package drone implements output to .drone.yml.
package drone

import (
	"bytes"
	"io"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/pretty"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".drone.yml"
)

// Output implements Drone project config generation.
type Output struct { //nolint:govet
	output.FileAdapter

	manifest *yaml.Manifest

	defaultPipeline *yaml.Pipeline
	notifyPipeline  *yaml.Pipeline

	standardMounts  []*yaml.VolumeMount
	standardVolumes []*yaml.Volume

	PipelineType       string
	NotifySlackChannel string
	BuildContainer     string
}

// NewOutput creates new .drone.yml output.
func NewOutput() *Output {
	output := &Output{
		manifest: &yaml.Manifest{},

		PipelineType:       "kubernetes",
		NotifySlackChannel: "proj-talos-maintainers",
		BuildContainer:     buildContainer,
	}

	output.standardMounts = []*yaml.VolumeMount{}
	output.standardVolumes = []*yaml.Volume{}

	output.defaultPipeline = &yaml.Pipeline{
		Name: "default",
		Type: output.PipelineType,
		Kind: "pipeline",
	}

	output.defaultPipeline.Trigger = yaml.Conditions{
		Branch: yaml.Condition{
			Exclude: []string{
				"renovate/*",
				"dependabot/*",
			},
		},
	}

	output.notifyPipeline = &yaml.Pipeline{
		Name: "notify",
		Type: output.PipelineType,
		Kind: "pipeline",
		Clone: yaml.Clone{
			Disable: true,
		},
		Trigger: yaml.Conditions{
			Status: yaml.Condition{
				Include: []string{"success", "failure"},
			},
		},
		Steps: []*yaml.Container{
			{
				Name:  "slack",
				Image: "plugins/slack",
				Settings: map[string]*yaml.Parameter{
					"webhook": {
						Secret: "slack_webhook",
					},
					"channel": {
						Value: output.NotifySlackChannel,
					},
					"link_names": {
						Value: true,
					},
					"template": {
						Value: notifyTemplate,
					},
				},
				When: yaml.Conditions{
					Status: yaml.Condition{
						Include: []string{"success", "failure"},
					},
				},
			},
		},
		DependsOn: []string{"default"},
	}

	output.manifest.Resources = append(output.manifest.Resources, output.defaultPipeline, output.notifyPipeline)

	output.FileAdapter.FileWriter = output

	return output
}

// Step appends a step to the default pipeline.
func (o *Output) Step(step *Step) {
	if step.container.Image == "" {
		step.container.Image = o.BuildContainer
	}

	if step.container.Pull == "" {
		step.container.Pull = "always"
	}

	step.container.Volumes = append(step.container.Volumes, o.standardMounts...)

	o.defaultPipeline.Steps = append(o.defaultPipeline.Steps, &step.container)
}

// Compile implements output.Writer interface.
func (o *Output) Compile(node interface{}) error {
	compiler, implements := node.(Compiler)
	if !implements {
		return nil
	}

	return compiler.CompileDrone(o)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	return []string{filename}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.drone(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) drone(w io.Writer) error {
	// fix up volumes
	o.defaultPipeline.Volumes = o.standardVolumes

	preamble := output.Preamble("# ")

	var buf bytes.Buffer

	pretty.Print(&buf, o.manifest)

	firstLine, err := buf.ReadString('\n')
	if err != nil {
		return err
	}

	if _, err := w.Write([]byte(firstLine)); err != nil {
		return err
	}

	if _, err := w.Write([]byte(preamble)); err != nil {
		return err
	}

	if _, err := buf.WriteTo(w); err != nil {
		return err
	}

	return nil
}

// Compiler is implemented by project blocks which support Drone config generation.
type Compiler interface {
	CompileDrone(*Output) error
}

// CustomCompiler is implemented by custom steps.
type CustomCompiler interface {
	DroneEnabled() bool
}

// HasDroneOutput checks if the node implements Compiler and has any output from drone.
func HasDroneOutput() dag.NodeCondition {
	return func(node dag.Node) bool {
		if !dag.Implements[*Compiler]()(node) {
			return false
		}

		if c, ok := node.(CustomCompiler); ok && !c.DroneEnabled() {
			return false
		}

		return true
	}
}
