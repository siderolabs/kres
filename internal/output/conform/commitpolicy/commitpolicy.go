// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package commitpolicy contains YAML structures for commit policy in .conform.yaml.
//
//nolint:govet
package commitpolicy

// Policy represents a commit policy.
type Policy struct {
	Type string `yaml:"type"`
	Spec Spec   `yaml:"spec"`
}

// Spec represents a commit policy spec in a commit policy.
type Spec struct {
	DCO                bool         `yaml:"dco"`
	GPG                GPG          `yaml:"gpg"`
	Spellcheck         Spellcheck   `yaml:"spellcheck"`
	MaximumOfOneCommit bool         `yaml:"maximumOfOneCommit"`
	Header             Header       `yaml:"header"`
	Body               Body         `yaml:"body"`
	Conventional       Conventional `yaml:"conventional"`
}

// GPG represents a GPG policy in a commit policy spec.
type GPG struct {
	Required bool     `yaml:"required"`
	Identity Identity `yaml:"identity,omitempty"`
}

// Identity represents an identity in a GPG config.
type Identity struct {
	GithubOrganization string `yaml:"gitHubOrganization,omitempty"`
}

// Spellcheck represents a spellcheck policy in a commit policy spec.
type Spellcheck struct {
	Locale string `yaml:"locale"`
}

// Header represents a header policy in a commit policy spec.
type Header struct {
	Length                int    `yaml:"length"`
	Imperative            bool   `yaml:"imperative"`
	Case                  string `yaml:"case"`
	InvalidLastCharacters string `yaml:"invalidLastCharacters"`
}

// Body represents a body policy in a commit policy spec.
type Body struct {
	Required bool `yaml:"required"`
}

// Conventional represents a conventional commit policy in a commit policy spec.
type Conventional struct {
	Types  []string `yaml:"types"`
	Scopes []string `yaml:"scopes"`
}

// New creates a new commit policy.
func New(organization string, enableGPGSignatureCheck bool, types, scopes []string, maximumOfOneCommit bool) Policy {
	return Policy{
		Type: "commit",
		Spec: Spec{
			DCO:                true,
			GPG:                GPG{Required: enableGPGSignatureCheck, Identity: Identity{GithubOrganization: organization}},
			Spellcheck:         Spellcheck{Locale: "US"},
			MaximumOfOneCommit: maximumOfOneCommit,
			Header:             Header{Length: 89, Imperative: true, Case: "lower", InvalidLastCharacters: "."},
			Body:               Body{Required: true},
			Conventional:       Conventional{Types: types, Scopes: scopes},
		},
	}
}
