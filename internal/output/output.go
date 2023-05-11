// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package output defines basic output features.
package output

// Writer is an interface which should be implemented by outputs.
type Writer interface {
	Generate() error
	Compile(any) error
}

// TypedWriter is an interface which should be implemented by outputs. It is a typed version of Writer.
type TypedWriter[T any] interface {
	Generate() error
	Compile(T) error
}

type adapter[T any] struct {
	inner TypedWriter[T]
}

func (w *adapter[T]) Generate() error { return w.inner.Generate() }

func (w *adapter[T]) Compile(i any) error {
	val, ok := i.(T)
	if !ok {
		return nil
	}

	return w.inner.Compile(val)
}

// Wrap creates a [Writer] instance from a [TypedWriter] instance.
func Wrap[T any](w TypedWriter[T]) Writer {
	return &adapter[T]{inner: w}
}
