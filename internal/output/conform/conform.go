// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package conform implements output to .conform.yaml.
package conform

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/siderolabs/kres/internal/output"
	"github.com/siderolabs/kres/internal/output/conform/commitpolicy"
	"github.com/siderolabs/kres/internal/output/conform/licensepolicy"
)

const (
	filename = ".conform.yaml"
)

// Output implements .conform.yaml generation.
type Output struct {
	output.FileAdapter

	githubOrg          string
	scopes             []string
	types              []string
	licensePolicySpecs []licensepolicy.Spec
	maxOfOneCommit     bool

	licenseCheck      bool
	gpgSignatureCheck bool

	enabled bool
}

// NewOutput creates new conform.yaml output.
func NewOutput() *Output {
	output := &Output{}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileConform(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// SetScopes sets the conventional commit scopes.
func (o *Output) SetScopes(scopes []string) {
	o.scopes = scopes
}

// SetTypes sets the conventional commit types.
func (o *Output) SetTypes(types []string) {
	o.types = types
}

// SetLicenseCheck enables license check.
func (o *Output) SetLicenseCheck(enable bool) {
	o.licenseCheck = enable
}

// SetLicensePolicySpecs sets the license policy specs.
func (o *Output) SetLicensePolicySpecs(licensePolicySpecs []licensepolicy.Spec) {
	o.licensePolicySpecs = licensePolicySpecs
}

// SetGPGSignatureCheck enables GPG signature check.
func (o *Output) SetGPGSignatureCheck(enable bool) {
	o.gpgSignatureCheck = enable
}

// SetMaximumOfOneCommit enables single commit check.
func (o *Output) SetMaximumOfOneCommit(enable bool) {
	o.maxOfOneCommit = enable
}

// SetGitHubOrganization scopes GPG identity check to the GitHub organization.
func (o *Output) SetGitHubOrganization(org string) {
	o.githubOrg = org
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return []string{filename}
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(filename string, w io.Writer) error {
	switch filename {
	case filename:
		return o.config(w)
	default:
		panic("unexpected filename: " + filename)
	}
}

func (o *Output) config(w io.Writer) error {
	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	policyList := []any{
		commitpolicy.New(o.githubOrg, o.gpgSignatureCheck, o.types, o.scopes, o.maxOfOneCommit),
	}

	for _, spec := range o.licensePolicySpecs {
		policyList = append(policyList, licensepolicy.New(spec))
	}

	policies := map[string]any{
		"policies": policyList,
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)

	if err := encoder.Encode(policies); err != nil {
		return fmt.Errorf("failed to encode policies: %w", err)
	}

	return nil
}

// Compiler is implemented by project blocks which support .conform.yaml generate.
type Compiler interface {
	CompileConform(*Output) error
}
