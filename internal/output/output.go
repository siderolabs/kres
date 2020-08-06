// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package output defines basic output features.
package output

// Writer is an interface which should be implemented by outputs.
type Writer interface {
	Generate() error
	Compile(interface{}) error
}
