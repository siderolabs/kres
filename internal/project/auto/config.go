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
	Name       string   `yaml:"name"`
	Inputs     []string `yaml:"inputs"`
	Dependants []string `yaml:"dependants"`
	Toplevel   bool     `yaml:"toplevel"`
}

// CI defines CI settings.
type CI struct {
	Provider string `yaml:"provider"`
	// CompileGHWorkflowsOnly is a flag to generate only GitHub Actions.
	CompileGHWorkflowsOnly bool `yaml:"compileGHWorkflowsOnly"`
}

// Helm defines helm settings.
type Helm struct {
	ChartDir string       `yaml:"chartDir"`
	E2EDir   string       `yaml:"e2eDir"`
	Template HelmTemplate `yaml:"template"`
	Enabled  bool         `yaml:"enabled"`
}

// HelmTemplate defines helm template settings.
type HelmTemplate struct {
	Set        []string `yaml:"set"`
	SetFile    []string `yaml:"setFile"`
	SetJSON    []string `yaml:"setJSON"`
	SetLiteral []string `yaml:"setLiteral"`
	SetString  []string `yaml:"setString"`
}

// IntegrationTests defines integration tests builder to be generated.
type IntegrationTests struct {
	Tests []IntegrationTestConfig `yaml:"tests"`
}

// IntegrationTestConfig defines the integration tests build configuration.
type IntegrationTestConfig struct {
	Outputs           map[string]map[string]string `yaml:"outputs"`
	Name              string                       `yaml:"name"`
	Path              string                       `yaml:"path"`
	ImageName         string                       `yaml:"imageName"`
	EnableDockerImage bool                         `yaml:"enableDockerImage"`
}
