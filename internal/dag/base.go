// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dag

import "slices"

// BaseNode implements core functionality of the node.
//
// BaseNode is designed to be included into other types.
type BaseNode struct { //nolint:govet
	parents []Node
	inputs  []Node
	name    string
}

// NewBaseNode creates new embeddable BaseNode.
func NewBaseNode(name string) BaseNode {
	return BaseNode{
		name: name,
	}
}

// Name implements Node interface.
func (node *BaseNode) Name() string {
	return node.name
}

// Inputs implements Node interface.
func (node *BaseNode) Inputs() []Node {
	return node.inputs
}

// Parents implements Node interface.
func (node *BaseNode) Parents() []Node {
	return node.parents
}

// AddParent implements Node interface.
func (node *BaseNode) AddParent(parent Node) {
	if !slices.Contains(node.parents, parent) {
		node.parents = append(node.parents, parent)
	}
}

// AddInput implements Node interface.
func (node *BaseNode) AddInput(input ...Node) {
	for _, n := range input {
		n.AddParent(node)
	}

	node.inputs = append(node.inputs, input...)
}

// BaseGraph implements core functionality of DAG.
//
// BaseGraph is designed to be embedded into other types.
type BaseGraph struct {
	targets []Node
}

// Targets returns list of targets for the graph.
func (graph *BaseGraph) Targets() []Node {
	return graph.targets
}

// AddTarget adds new targets to the graph.
func (graph *BaseGraph) AddTarget(target ...Node) {
	graph.targets = append(graph.targets, target...)
}
