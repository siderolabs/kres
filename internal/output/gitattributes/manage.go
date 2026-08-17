// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gitattributes

import (
	"github.com/siderolabs/kres/internal/output"
)

// filenamer is implemented by outputs which report the files they write.
type filenamer interface {
	Filenames() []string
}

// managedFilenamer is implemented by outputs which need to report a narrower set of files than
// Filenames(), e.g. template.Output, which excludes NoOverwrite files: those are only created
// once on project initialization and shouldn't be marked as generated. When a source implements
// both interfaces, managedFilenamer takes precedence.
type managedFilenamer interface {
	ManagedFilenames() []string
}

// Manage returns w unchanged, except its Generate() call now also registers its filenames with
// output, marking them as generated. The result should still be adapted to
// output.Writer via output.Wrap, same as any other TypedWriter.
//
// output must be generated (via its own Generate() call, appended to the outputs
// list) after every Manage()-wrapped output, since each only reports its filenames as part of its
// own Generate() call.
func Manage[T any](output *Output, writer output.TypedWriter[T]) output.TypedWriter[T] {
	return &trackingWriter[T]{
		output,
		writer,
	}
}

type trackingWriter[T any] struct {
	output *Output
	writer output.TypedWriter[T]
}

func (t *trackingWriter[T]) Compile(v T) error {
	return t.writer.Compile(v)
}

func (t *trackingWriter[T]) Generate() error {
	switch source := t.writer.(type) {
	case managedFilenamer:
		t.output.MarkGenerated(source.ManagedFilenames()...)
	case filenamer:
		t.output.MarkGenerated(source.Filenames()...)
	}

	return t.writer.Generate()
}
