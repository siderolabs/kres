// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package dag represents abstract directed acyclic graph.
package dag

// Graph represents the targets of the build process.
type Graph interface {
	Targets() []Node
}

// WalkFunc is a callback function called by Walk.
type WalkFunc func(node Node) error

// Walk the graph calling function for every node just once.
func Walk(graph Graph, walkFn WalkFunc, visited map[Node]struct{}) error {
	if visited == nil {
		visited = make(map[Node]struct{})
	}

	targets := graph.Targets()

	return walk(targets, walkFn, visited)
}

// WalkNode walks the graph starting from the node just once.
func WalkNode(node Node, walkFn WalkFunc, visited map[Node]struct{}) error {
	if visited == nil {
		visited = make(map[Node]struct{})
	}

	targets := node.Inputs()

	return walk(targets, walkFn, visited)
}

func walk(targets []Node, walkFn WalkFunc, visited map[Node]struct{}) error {
	for _, target := range targets {
		if _, ok := visited[target]; ok {
			continue
		}

		visited[target] = struct{}{}

		if err := walk(target.Inputs(), walkFn, visited); err != nil {
			return err
		}

		if err := walkFn(target); err != nil {
			return err
		}
	}

	return nil
}
