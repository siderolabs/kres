// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package command

import "github.com/mitchellh/cli"

// Meta is a shared command functionality.
type Meta struct {
	Ui cli.Ui //nolint: golint,stylecheck,revive
}
