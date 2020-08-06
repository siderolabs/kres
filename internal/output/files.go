// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package output

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
)

// FileWriter interface can be adapted to Writer interface via FileAdapter.
type FileWriter interface {
	Filenames() []string
	GenerateFile(filename string, w io.Writer) error
}

// FileAdapter implements Writer via FileWriter.
type FileAdapter struct {
	FileWriter
}

// Generate implements outout.Writer.
func (adapter *FileAdapter) Generate() error {
	// buffer the output before writing it down
	buffers := map[string]*bytes.Buffer{}

	for _, filename := range adapter.FileWriter.Filenames() {
		buf := bytes.NewBuffer(nil)

		if err := adapter.FileWriter.GenerateFile(filename, buf); err != nil {
			return err
		}

		buffers[filename] = buf
	}

	// write everything back to the filesystem
	for _, filename := range adapter.FileWriter.Filenames() {
		filename := filename

		var oldContents []string

		if err := func() error {
			f, err := os.Open(filename)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}

				return err
			}

			defer f.Close() //nolint: errcheck

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

			defer f.Close() //nolint: errcheck

			_, err = buffers[filename].WriteTo(f)

			return err
		}(); err != nil {
			return err
		}
	}

	return nil
}

func splitIgnoringPreamble(r io.Reader) ([]string, error) {
	var contents []string

	inPreamble := true

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if inPreamble && (strings.HasPrefix(line, "#") || line == "") { // comments, skip it as might be a preamble
			continue
		}

		inPreamble = false

		contents = append(contents, line)
	}

	return contents, scanner.Err()
}
