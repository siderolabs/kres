// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package licensepolicy contains the YAML structure for license policy in .conform.yaml.
//
//nolint:govet
package licensepolicy

// LicensePolicy represents a license policy.
type LicensePolicy struct {
	Type string `yaml:"type"`
	Spec Spec   `yaml:"spec"`
}

// Spec represents a license policy spec in a license policy.
type Spec struct {
	Root            string   `yaml:"root,omitempty"`
	SkipPaths       []string `yaml:"skipPaths"`
	IncludeSuffixes []string `yaml:"includeSuffixes"`
	ExcludeSuffixes []string `yaml:"excludeSuffixes"`
	Header          string   `yaml:"header"`
}

func (l *Spec) initDefaults() {
	if l.SkipPaths == nil {
		l.SkipPaths = []string{
			".git/",
			"testdata/",
		}
	}

	if l.IncludeSuffixes == nil {
		l.IncludeSuffixes = []string{
			".go",
		}
	}

	if l.ExcludeSuffixes == nil {
		l.ExcludeSuffixes = []string{
			".pb.go",
			".pb.gw.go",
		}
	}
}

// New creates a new license policy.
func New(spec Spec) LicensePolicy {
	spec.initDefaults()

	return LicensePolicy{
		Type: "license",
		Spec: spec,
	}
}
