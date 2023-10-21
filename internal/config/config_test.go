// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/kres/internal/config"
)

func TestMissingFile(t *testing.T) {
	provider, err := config.NewProvider("testdata/nosuchfile.yaml")
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

type Foo struct { //nolint:govet
	Contents string
	Len      int
	Extra    string

	name string
}

func (foo *Foo) Name() string {
	return foo.name
}

type Other struct{}

func TestLoad(t *testing.T) {
	provider, err := config.NewProvider("testdata/.kres.yaml")
	require.NoError(t, err)

	foo := Foo{
		name: "Bar",
	}

	err = provider.Load(&foo)
	require.NoError(t, err)

	assert.Equal(t, "xyz", foo.Contents)
	assert.Equal(t, 5, foo.Len)
	assert.Equal(t, "same", foo.Extra)

	bad := Foo{
		name: "Bad",
	}
	err = provider.Load(&bad)
	require.NoError(t, err)

	err = provider.Load(&Other{})
	require.EqualError(t, err, "config has name blah for kind config_test.Other, while object doesn't support names")

	reallyBad := Foo{
		name: "ReallyBad",
	}
	err = provider.Load(&reallyBad)
	require.EqualError(t, err, "error decoding config block config_test.Foo/ReallyBad into &{ 0 same ReallyBad}: yaml: unmarshal errors:\n  line 32: cannot unmarshal !!str `infinite` into int")

	noSpec := Foo{
		name: "NoSpec",
	}
	err = provider.Load(&noSpec)
	require.EqualError(t, err, "missing spec for config block config_test.Foo/NoSpec")
}
