// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package golang

import (
	"os"
	"strings"
)

func genName(name, projectPath string) string {
	if projectPath == "." || projectPath == "" {
		return name
	}

	return strings.Join(append([]string{name}, strings.Split(projectPath, string(os.PathSeparator))...), "-")
}
