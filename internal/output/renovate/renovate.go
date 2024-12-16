// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package renovate implements output to .github/renovate.json
package renovate

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/siderolabs/kres/internal/output"
)

// Output provides output to .github/renovate.json.
type Output struct {
	output.FileAdapter

	result *Renovate

	enabled bool
}

const (
	fileName = ".github/renovate.json"
)

// NewOutput initializes Output.
func NewOutput() *Output {
	o := &Output{}

	o.FileWriter = o

	preamble := output.Preamble("")

	o.result = &Renovate{
		Schema:      "https://docs.renovatebot.com/renovate-schema.json",
		Description: preamble,
		Extends: []string{
			":dependencyDashboard",
			":gitSignOff",
			":semanticCommitScopeDisabled",
			"schedule:earlyMondays",
		},
		PRHeader:           "Update Request | Renovate Bot",
		SeparateMajorMinor: false,
		PackageRules: []PackageRule{
			{
				MatchUpdateTypes: []string{
					"major",
					"minor",
					"patch",
					"pin",
					"digest",
				},
				GroupName: "dependencies",
			},
		},
	}

	return o
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileRenovate(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// CustomManagers sets custom managers.
func (o *Output) CustomManagers(customManagers []CustomManager) {
	o.result.CustomManagers = customManagers
}

// PackageRules sets package rules.
func (o *Output) PackageRules(packageRules []PackageRule) {
	o.result.PackageRules = slices.Concat(o.result.PackageRules, packageRules)
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return []string{fileName}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case fileName:
		return o.renovate(w)
	default:
		panic("unexpected filename: " + fileName)
	}
}

func (o *Output) renovate(w io.Writer) error {
	encoder := json.NewEncoder(w)

	encoder.SetIndent("", "    ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(o.result); err != nil {
		return fmt.Errorf("failed to encode renovate config: %w", err)
	}

	return nil
}

// Compiler is implemented by project blocks which support renovate config generation.
type Compiler interface {
	CompileRenovate(*Output) error
}
