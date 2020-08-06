// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package auto provides automatic detector of project type, reflections.
package auto

import (
	"os"
	"path/filepath"
)

func directoryExists(rootPath, name string) (bool, error) {
	path := filepath.Join(rootPath, name)

	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return st.IsDir(), nil
}
