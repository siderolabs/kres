// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package templates defines default templates for various Go components.
package templates

import _ "embed"

// VersionGo version.go
//
//go:embed version_go
var VersionGo string
