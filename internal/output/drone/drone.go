// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package drone implements output to .drone.yml.
package drone

import (
	"bytes"
	"io"
	"slices"

	"github.com/drone/drone-yaml/yaml"
	"github.com/drone/drone-yaml/yaml/pretty"

	"github.com/siderolabs/kres/internal/dag"
	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".drone.yml"
)

// StepService provides base Drone compilation to a pipeline.
type StepService interface {
	Service(spec *yaml.Container)
	Step(step *Step)
}

// Output implements Drone project config generation.
type Output struct { //nolint:govet
	output.FileAdapter

	manifest *yaml.Manifest

	defaultPipeline *yaml.Pipeline
	notifyPipeline  *yaml.Pipeline

	standardMounts []*yaml.VolumeMount
	volumes        []*yaml.Volume

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
	output.volumes = []*yaml.Volume{}

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
		Event: yaml.Condition{
			Exclude: []string{
				"promote",
				"cron",
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
			Branch: yaml.Condition{
				Exclude: []string{
					"renovate/*",
					"dependabot/*",
				},
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
	}

	output.manifest.Resources = append(output.manifest.Resources, output.defaultPipeline)

	output.FileAdapter.FileWriter = output

	return output
}

// Step appends a step to the default pipeline.
func (o *Output) Step(step *Step) {
	o.appendStep(step, o.defaultPipeline)
}

func (o *Output) appendStep(originalStep *Step, pipeline *yaml.Pipeline) {
	// perform a shallow copy of the step to avoid modifying the original
	step := *originalStep

	step.container.Volumes = slices.Clone(step.container.Volumes)

	if step.container.Image == "" {
		step.container.Image = o.BuildContainer
	}

	if step.container.Pull == "" {
		step.container.Pull = "always"
	}

	step.container.Volumes = append(step.container.Volumes, o.standardMounts...)

	for _, volume := range step.volumes {
		if !slices.ContainsFunc(o.volumes, func(v *yaml.Volume) bool { return v.Name == volume.Name }) {
			o.volumes = append(o.volumes, step.volumes...)
		}
	}

	pipeline.Steps = append(pipeline.Steps, &step.container)
}

// Pipeline creates a new pipeline which can be triggered via promotion/cron.
func (o *Output) Pipeline(name string, targets []string, crons []string) *Pipeline {
	p := &Pipeline{
		drone: o,
	}

	if len(targets) > 0 {
		targetPipeline := &yaml.Pipeline{
			Name: name,
			Type: o.PipelineType,
			Kind: "pipeline",
			Trigger: yaml.Conditions{
				Target: yaml.Condition{
					Include: targets,
				},
			},
		}

		o.manifest.Resources = append(o.manifest.Resources, targetPipeline)
		p.pipelines = append(p.pipelines, targetPipeline)
	}

	if len(crons) > 0 {
		cronPipeline := &yaml.Pipeline{
			Name: "cron-" + name,
			Type: o.PipelineType,
			Kind: "pipeline",
			Trigger: yaml.Conditions{
				Cron: yaml.Condition{
					Include: crons,
				},
			},
		}

		o.manifest.Resources = append(o.manifest.Resources, cronPipeline)
		p.pipelines = append(p.pipelines, cronPipeline)
	}

	return p
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
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
	for _, r := range o.manifest.Resources {
		if pipeline, ok := r.(*yaml.Pipeline); ok {
			pipeline.Volumes = o.volumes
		}
	}

	// fix up notify pipeline
	for _, r := range o.manifest.Resources {
		if pipeline, ok := r.(*yaml.Pipeline); ok {
			o.notifyPipeline.DependsOn = append(o.notifyPipeline.DependsOn, pipeline.Name)
		}
	}

	o.manifest.Resources = append(o.manifest.Resources, o.notifyPipeline)

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
		if !dag.Implements[Compiler]()(node) {
			return false
		}

		if c, ok := node.(CustomCompiler); ok && !c.DroneEnabled() {
			return false
		}

		return true
	}
}
