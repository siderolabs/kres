// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package golangci implements output to .golangci.yml.
package golangci

import (
	_ "embed"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/siderolabs/gen/xslices"
	"go.yaml.in/yaml/v4"

	"github.com/siderolabs/kres/internal/output"
)

const (
	filename = ".golangci.yml"
)

//go:embed golangci.yml
var baseConfig []byte

// Output implements .golangci.yml generation.
type Output struct {
	output.FileAdapter

	files   []file
	enabled bool
}

type file struct {
	path    string
	configs []yaml.Node
}

// NewOutput creates new Makefile output.
func NewOutput() *Output {
	output := &Output{}

	output.FileWriter = output

	return output
}

// Compile implements [output.TypedWriter] interface.
func (o *Output) Compile(compiler Compiler) error {
	return compiler.CompileGolangci(o)
}

// Enable should be called to enable config generation.
func (o *Output) Enable() {
	o.enabled = true
}

// NewFile adds the config of a project, with the configuration fragments to merge into the base configuration, in order.
func (o *Output) NewFile(path string, configs ...yaml.Node) {
	o.files = append(o.files, file{
		path:    filepath.Join(path, filename),
		configs: configs,
	})
}

// Filenames implements output.FileWriter interface.
func (o *Output) Filenames() []string {
	if !o.enabled {
		return nil
	}

	return xslices.Map(o.files, func(f file) string { return f.path })
}

// GenerateFile implements output.FileWriter interface.
func (o *Output) GenerateFile(path string, w io.Writer) error {
	idx := slices.IndexFunc(o.files, func(f file) bool { return f.path == path })
	if idx < 0 {
		return fmt.Errorf("unexpected file %q", path)
	}

	configs := o.files[idx].configs

	if _, err := w.Write([]byte(output.Preamble("# "))); err != nil {
		return err
	}

	var doc yaml.Node

	if err := yaml.Unmarshal(baseConfig, &doc); err != nil {
		return fmt.Errorf("error parsing the base golangci-lint config: %w", err)
	}

	root := doc.Content[0]

	for i := range configs {
		merge(root, &configs[i])
	}

	if len(configs) > 0 {
		enableLinters(root)
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)

	if err := encoder.Encode(&doc); err != nil {
		return err
	}

	return encoder.Close()
}

// merge merges src into dst: mappings are merged key by key, sequences are appended, anything else is replaced.
func merge(dst, src *yaml.Node) {
	switch {
	case dst.Kind == yaml.MappingNode && src.Kind == yaml.MappingNode:
		for i := 0; i+1 < len(src.Content); i += 2 {
			key, value := src.Content[i], src.Content[i+1]

			if existing := lookup(dst, key.Value); existing != nil {
				merge(existing, value)
			} else {
				dst.Content = append(dst.Content, key, value)
			}
		}

		dst.Style = 0 // an empty flow mapping ({}) which received keys is rendered as a block
	case dst.Kind == yaml.SequenceNode && src.Kind == yaml.SequenceNode:
		dst.Content = append(dst.Content, src.Content...)
		dst.Style = 0
	default:
		*dst = *src
	}
}

// enableLinters removes the linters listed under linters.enable from linters.disable.
//
// The base config enables all linters by default and disables some of them, and golangci-lint silently
// keeps a linter disabled if it is listed in both, so enabling one means taking it off the disabled list.
func enableLinters(root *yaml.Node) {
	linters := lookup(root, "linters")
	if linters == nil {
		return
	}

	enable, disable := lookup(linters, "enable"), lookup(linters, "disable")
	if enable == nil || disable == nil {
		return
	}

	enabled := xslices.Map(enable.Content, func(n *yaml.Node) string { return n.Value })

	disable.Content = slices.DeleteFunc(disable.Content, func(n *yaml.Node) bool {
		return slices.Contains(enabled, n.Value)
	})
}

// lookup returns the value node for the key in the mapping node, or nil.
func lookup(mapping *yaml.Node, key string) *yaml.Node {
	if mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

// Compiler is implemented by project blocks which support Dockerfile generate.
type Compiler interface {
	CompileGolangci(*Output) error
}
