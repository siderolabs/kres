// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dag

import (
	"slices"

	"github.com/siderolabs/gen/xslices"
)

// Node in directed acyclic graph, recording parent nodes as inputs.
type Node interface {
	Name() string
	Inputs() []Node
	AddInput(...Node)
}

// NodeCondition checks the node for a specific condition.
type NodeCondition func(Node) bool

// Implements checks whether node implements specific type T.
func Implements[T any]() NodeCondition {
	return func(node Node) bool {
		_, ok := node.(T)

		return ok
	}
}

// Not inverts the check.
func Not(condition NodeCondition) NodeCondition {
	return func(node Node) bool {
		return !condition(node)
	}
}

// GatherMatchingInputNames scans all the inputs and returns those which match the condition.
func GatherMatchingInputNames(node Node, condition NodeCondition) []string {
	return xslices.Map(GatherMatchingInputs(node, condition), func(input Node) string {
		return input.Name()
	})
}

// GatherMatchingInputs scans all the inputs and returns those which match the condition.
func GatherMatchingInputs(node Node, condition NodeCondition) []Node {
	return xslices.Filter(node.Inputs(), func(input Node) bool { return condition(input) })
}

// GatherMatchingInputsRecursive scans all the inputs recursively and returns those which match the condition.
func GatherMatchingInputsRecursive(node Node, condition NodeCondition) []Node {
	result := GatherMatchingInputs(node, condition)

	for _, input := range node.Inputs() {
		downstream := GatherMatchingInputsRecursive(input, condition)

		for _, downstreamInput := range downstream {
			if !slices.Contains(result, downstreamInput) {
				result = append(result, downstreamInput)
			}
		}
	}

	return result
}
