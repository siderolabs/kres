// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package config provides config loading and mapping.
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Document is a part of config.
type Document struct {
	// Class name and package name, e.g. `golang.Toolchain`.
	Kind string `yaml:"kind"`

	// Name of particular object (if supported).
	Name string `yaml:"name,omitempty"`

	// Spec is loaded into the matching object.
	Spec yaml.Node `yaml:"spec"`
}

// Provider resolves configuration for each object.
type Provider struct {
	docs []Document
}

// NewProvider loads configuration file and parses it.
func NewProvider(path string) (*Provider, error) {
	r, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Provider{}, nil
		}

		return nil, err
	}

	defer r.Close() //nolint:errcheck

	decoder := yaml.NewDecoder(r)

	provider := &Provider{}

	for {
		var doc Document

		if err := decoder.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return provider, err
		}

		provider.docs = append(provider.docs, doc)
	}

	decoder.KnownFields(true)

	return provider, nil
}

// Load config into passed object.
//
// All the matching configs are loaded in the order specified.
func (provider *Provider) Load(obj any) error {
	type named interface {
		Name() string
	}

	typ := reflect.TypeOf(obj).Elem()
	kind := path.Base(typ.PkgPath()) + "." + typ.Name()

	name := ""
	if namedObj, ok := obj.(named); ok {
		name = namedObj.Name()
	}

	for _, doc := range provider.docs {
		if doc.Kind != kind {
			continue
		}

		if doc.Name != "" && name == "" {
			return fmt.Errorf("config has name %v for kind %v, while object doesn't support names", doc.Name, kind)
		}

		if doc.Name == "" || doc.Name == name {
			if doc.Spec.IsZero() {
				return fmt.Errorf("missing spec for config block %v/%v", doc.Kind, doc.Name)
			}

			if err := doc.Spec.Decode(obj); err != nil {
				return fmt.Errorf("error decoding config block %v/%v into %v: %w", doc.Kind, doc.Name, obj, err)
			}
		}
	}

	return nil
}
