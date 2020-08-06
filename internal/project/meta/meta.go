// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package meta provides project options from source code.
package meta

// Options for the project.
type Options struct {
	// CanonicalPath, import path for Go projects.
	CanonicalPath string

	// VersionPackage is a canonical path to version package (if any).
	VersionPackage string

	// Directories which contain source code.
	Directories []string

	// Source files on top level.
	SourceFiles []string

	// BuildArgs passed down to Dockerfiles.
	BuildArgs []string

	// Path to /bin.
	BinPath string

	// Path to ~/.cache.
	CachePath string
}
