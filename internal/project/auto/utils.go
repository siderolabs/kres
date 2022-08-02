// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"os"
	"path/filepath"
	"strings"
)

func listFilesWithSuffix(path, suffix string) ([]string, error) {
	contents, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []string

	for _, item := range contents {
		if !item.IsDir() && strings.HasSuffix(item.Name(), suffix) {
			result = append(result, item.Name())
		}
	}

	return result, nil
}

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
