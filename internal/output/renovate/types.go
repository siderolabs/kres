// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package renovate

// Renovate represents the renovate configuration.
type Renovate struct {
	Schema             string          `json:"$schema"`
	Description        string          `json:"description"`
	PRHeader           string          `json:"prHeader"`
	Extends            []string        `json:"extends"`
	CustomManagers     []CustomManager `json:"customManagers,omitempty"`
	PackageRules       []PackageRule   `json:"packageRules,omitempty"`
	SeparateMajorMinor bool            `json:"separateMajorMinor"`
}

// CustomManager represents a custom manager.
type CustomManager struct {
	CustomType         string   `json:"customType"`
	DataSourceTemplate string   `json:"datasourceTemplate,omitempty"`
	DepNameTemplate    string   `json:"depNameTemplate,omitempty"`
	VersioningTemplate string   `json:"versioningTemplate"`
	FileMatch          []string `json:"fileMatch"`
	MatchStrings       []string `json:"matchStrings"`
}

// PackageRule represents a package rule.
type PackageRule struct {
	Enabled            *bool  `json:"enabled,omitempty"`
	AllowedVersions    string `json:"allowedVersions,omitempty"`
	DataSourceTemplate string `json:"datasourceTemplate,omitempty"`
	DepNameTemplate    string `json:"depNameTemplate,omitempty"`
	GroupName          string `json:"groupName,omitempty"`
	Versioning         string `json:"versioning,omitempty"`
	VersioningTemplate string `json:"versioningTemplate,omitempty"`

	MatchDataSources []string `json:"matchDataSources,omitempty"`
	MatchFileNames   []string `json:"matchFileNames,omitempty"`
	MatchPaths       []string `json:"matchPaths,omitempty"`

	MatchPackageNames []string `json:"matchPackageNames,omitempty"`
	MatchUpdateTypes  []string `json:"matchUpdateTypes,omitempty"`
}
