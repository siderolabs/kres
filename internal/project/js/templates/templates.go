// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package templates defines default templates for various JS components.
package templates

import (
	_ "embed"
)

// Babel babel.config.js.
//
//go:embed babel.js
var Babel string

// Eslint eslintrc.yaml.
//
//go:embed eslint.yaml
var Eslint string

// Jest jest.config.js.
//
//go:embed jest.js
var Jest string

// TSConfig tsconfig.json.
//
//go:embed tsconfig.json
var TSConfig string

// GoEmbed is a complimentary file that is generated for each JS distribution.
// it allows embedding the data into a Go service.
var GoEmbed = `package {{.project}}

import "embed"

// Dist is an embedded JS frontend release folder.
//
//go:embed dist
var Dist embed.FS
`
