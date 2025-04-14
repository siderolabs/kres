// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package output

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileWriter interface can be adapted to Writer interface via FileAdapter.
type FileWriter interface {
	Filenames() []string
	GenerateFile(filename string, w io.Writer) error
}

// FilePermissionsWriter defines the requirements for setting the file
// permissions of a FileWriter. This interface is optional.
type FilePermissionsWriter interface {
	Permissions(filename string) os.FileMode
}

// FileAdapter implements Writer via FileWriter.
type FileAdapter struct {
	FileWriter
}

// ErrSkip makes file adapter skip the file write.
var ErrSkip = errors.New("skip file")

// Generate implements outout.Writer.
//
//nolint:gocognit
func (adapter *FileAdapter) Generate() error {
	// buffer the output before writing it down
	buffers := map[string]*bytes.Buffer{}

	for _, filename := range adapter.Filenames() {
		buf := bytes.NewBuffer(nil)

		dir := filepath.Dir(filename)

		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}

		if err := adapter.GenerateFile(filename, buf); err != nil {
			if errors.Is(err, ErrSkip) {
				continue
			}

			return err
		}

		buffers[filename] = buf
	}

	// write everything back to the filesystem
	for _, filename := range adapter.Filenames() {
		if _, ok := buffers[filename]; !ok {
			continue
		}

		var oldContents []string

		if err := func() error {
			f, err := os.Open(filename)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}

				return err
			}

			defer f.Close() //nolint:errcheck

			oldContents, err = splitIgnoringPreamble(f)

			return err
		}(); err != nil {
			return err
		}

		if newContents, err := splitIgnoringPreamble(bytes.NewReader(buffers[filename].Bytes())); err != nil {
			return err
		} else if strings.Join(oldContents, "\n") == strings.Join(newContents, "\n") {
			continue // skip as no changes
		}

		if err := func() error {
			f, err := os.Create(filename)
			if err != nil {
				return err
			}

			defer f.Close() //nolint:errcheck

			_, err = buffers[filename].WriteTo(f)

			return err
		}(); err != nil {
			return err
		}

		if permsWriter, implements := adapter.FileWriter.(FilePermissionsWriter); implements {
			perms := permsWriter.Permissions(filename)

			if perms == 0 {
				perms = 0o644
			}

			if err := os.Chmod(filename, perms); err != nil {
				return err
			}
		}
	}

	return nil
}

func splitIgnoringPreamble(r io.Reader) ([]string, error) {
	var (
		contents []string
		comments = []string{"#", "<!--"}
	)

	inPreamble := true

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// `# syntax = ...` is a special case, it's not a preamble
		if strings.HasPrefix(line, "# syntax") {
			contents = append(contents, line)
		}

		// `#!` is a shebang and not preamble
		if strings.HasPrefix(line, "#!") {
			contents = append(contents, line)
		}

		if inPreamble && (stringContainsPreamble(line, comments...) || line == "" || line == "---") {
			continue
		}

		inPreamble = false

		contents = append(contents, line)
	}

	return contents, scanner.Err()
}

func stringContainsPreamble(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}

	return false
}
