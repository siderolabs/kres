// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dag

import (
	"reflect"
)

// Node in directed acyclic graph, recording parent nodes as inputs.
type Node interface {
	Name() string
	Inputs() []Node
	InputNames() []string
	AddInput(...Node)
}

// NodeCondition checks the node for a specific condition.
type NodeCondition func(Node) bool

// Implements checks whether node implements specific interface.
func Implements(typ interface{}) NodeCondition {
	return func(node Node) bool {
		return reflect.TypeOf(node).Implements(reflect.TypeOf(typ).Elem())
	}
}

// Not inverts the check.
func Not(condition NodeCondition) NodeCondition {
	return func(node Node) bool {
		return !condition(node)
	}
}

// GatherMatchingInputNames scans all the inputs and returns those which match the condition.
//
// If direct input doesn't match a condition, search continues up until matching node is found.
func GatherMatchingInputNames(node Node, condition NodeCondition) []string {
	result := []string{}

	for _, input := range node.Inputs() {
		if condition(input) {
			result = append(result, input.Name())
		}
	}

	return result
}
