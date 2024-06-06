// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package templates defines default templates for various JS components.
package templates

import (
	_ "embed"
)

// Eslint eslint.config.js
//
//go:embed eslint.config.js
var Eslint string

// TSConfig tsconfig.json.
//
//go:embed tsconfig.json
var TSConfig string

// Bunfig bunfig.toml.
//
//go:embed bunfig.toml
var Bunfig string

// TestSetup test/setup.ts.
//
//go:embed setup-tests.ts
var TestSetup string

// GoEmbed is a complimentary file that is generated for each JS distribution.
// It allows embedding the data into a Go service.
var GoEmbed = `package {{.project}}

import "embed"

// Dist is an embedded JS frontend release folder.
//
//go:embed dist
var Dist embed.FS
`
