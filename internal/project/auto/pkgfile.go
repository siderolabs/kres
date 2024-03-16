// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package auto

import (
	"os"
	"path/filepath"

	"github.com/siderolabs/kres/internal/config"
	"github.com/siderolabs/kres/internal/project/pkgfile"
)

func (builder *builder) DetectPkgFile() (bool, error) {
	if _, err := os.Stat(filepath.Join(builder.rootPath, config.ContainerImageFrontendPkgfile)); err == nil {
		return true, nil
	}

	return false, nil
}

func (builder *builder) BuildPkgFile() error {
	builder.meta.ContainerImageFrontend = config.ContainerImageFrontendPkgfile

	pkgfile := pkgfile.NewBuild(builder.meta)

	builder.targets = append(builder.targets, pkgfile)
	pkgfile.AddInput(builder.commonInputs...)

	return nil
}
