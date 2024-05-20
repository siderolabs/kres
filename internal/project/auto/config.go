// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

// NamedConfig is a base type which provides config name.
type NamedConfig struct {
	name string
}

// Name implements named interface.
func (cfg *NamedConfig) Name() string {
	return cfg.name
}

// CommandConfig sets up settings for command build.
type CommandConfig struct {
	NamedConfig

	DisableImage bool `yaml:"disableImage"`
}

// CustomSteps defines custom steps to be generated.
type CustomSteps struct {
	Steps []CustomStep `yaml:"steps"`
}

// CustomStep defines a custom step to be built.
type CustomStep struct {
	Name     string   `yaml:"name"`
	Inputs   []string `yaml:"inputs"`
	Toplevel bool     `yaml:"toplevel"`
}

// CI defines CI settings.
type CI struct {
	Provider string `yaml:"provider"`
	// CompileGHWorkflowsOnly is a flag to generate only GitHub Actions.
	CompileGHWorkflowsOnly bool `yaml:"compileGHWorkflowsOnly"`
}
