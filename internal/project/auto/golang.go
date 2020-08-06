// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/talos-systems/kres/internal/project/meta"
)

// DetectGolang check if project at rootPath is Go-based project.
func DetectGolang(rootPath string, options *meta.Options) (bool, error) {
	gomodPath := filepath.Join(rootPath, "go.mod")

	gomod, err := os.Open(gomodPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	defer gomod.Close() //nolint: errcheck

	contents, err := ioutil.ReadAll(gomod)
	if err != nil {
		return true, err
	}

	options.CanonicalPath = modfile.ModulePath(contents)

	for _, srcDir := range []string{"src", "internal", "pkg", "cmd"} {
		exists, err := directoryExists(rootPath, srcDir)
		if err != nil {
			return true, err
		}

		if exists {
			options.Directories = append(options.Directories, srcDir)
		}
	}

	options.SourceFiles = append(options.SourceFiles, "go.mod", "go.sum")

	for _, candidate := range []string{"pkg/version", "internal/version"} {
		exists, err := directoryExists(rootPath, candidate)
		if err != nil {
			return true, err
		}

		if exists {
			options.VersionPackage = path.Join(options.CanonicalPath, candidate)
		}
	}

	return true, nil
}
