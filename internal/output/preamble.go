// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/talos-systems/kres/internal/version"
)

// PreambleTimestamp marks the time files are generated.
var PreambleTimestamp time.Time

// PreambleCreator is the name of the program.
var PreambleCreator string

// Preamble returns file auto-generated preamble with specified comment character.
func Preamble(commentPrefix string, commentPostFixes ...string) string {
	const preamble = `
THIS FILE WAS AUTOMATICALLY GENERATED, PLEASE DO NOT EDIT.

Generated on %s by %s.
`

	if PreambleTimestamp.IsZero() {
		PreambleTimestamp = time.Now().UTC()
	}

	if PreambleCreator == "" {
		PreambleCreator = fmt.Sprintf("%s %s", version.Name, version.Tag)
	}

	preambleStr := fmt.Sprintf(preamble, PreambleTimestamp.Format(time.RFC3339), PreambleCreator)

	byLines := strings.Split(strings.TrimSpace(preambleStr), "\n")

	for i := range byLines {
		byLines[i] = strings.TrimSpace(commentPrefix + byLines[i] + strings.Join(commentPostFixes, " "))
	}

	return strings.Join(byLines, "\n") + "\n\n"
}

// License returns file auto-generated license.
func License(commentPrefix string) string {
	const license = `This Source Code Form is subject to the terms of the Mozilla Public
License, v. 2.0. If a copy of the MPL was not distributed with this
file, You can obtain one at http://mozilla.org/MPL/2.0/.
`

	byLines := strings.Split(strings.TrimSpace(license), "\n")
	for i := range byLines {
		byLines[i] = strings.TrimSpace(commentPrefix + byLines[i])
	}

	return strings.Join(byLines, "\n") + "\n\n"
}
