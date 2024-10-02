// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package toposort provides stable toposort implementation.
package toposort

// Most of this implementation is based on github.com/SOF3/go-stable-toposort

import (
	"sort"
)

// Node is a node.
type Node[T any] interface {
	Before(other T) bool
}

type (
	nodeNumber = int
	edgeNumber = int
	edge       [2]nodeNumber
)

type edgeIndex struct {
	index [2]map[nodeNumber]map[nodeNumber]edgeNumber
	slice []edge
}

func newEdgeIndex() *edgeIndex {
	index := &edgeIndex{}

	for i := range index.index {
		index.index[i] = map[nodeNumber]map[nodeNumber]edgeNumber{}
	}

	return index
}

func (index *edgeIndex) add(edge edge) {
	number := len(index.slice)
	index.slice = append(index.slice, edge)

	for pos := range index.index {
		if _, exists := index.index[pos][edge[pos]]; !exists {
			index.index[pos][edge[pos]] = map[nodeNumber]edgeNumber{}
		}

		index.index[pos][edge[pos]][edge[1-pos]] = number
	}
}

func (index *edgeIndex) removeIndex(edge edge) {
	for pos := range [...]int{0, 1} {
		delete(index.index[pos][edge[pos]], edge[1-pos])

		if len(index.index[pos][edge[pos]]) == 0 {
			delete(index.index[pos], edge[pos])
		}
	}
}

// Stable sorts nodes by Kahn's algorithm.
func Stable[N Node[N]](nodes []N) (output []N, cycle []N) {
	edges := newEdgeIndex()

	for i := range nodes {
		for j := i + 1; j < len(nodes); j++ {
			ij := nodes[i].Before(nodes[j])
			ji := nodes[j].Before(nodes[i])

			if ij && ji {
				return nil, []N{nodes[i], nodes[j]}
			}

			if ij {
				edges.add(edge{i, j})
			} else if ji {
				edges.add(edge{j, i})
			}
		}
	}

	output = make([]N, 0, len(nodes))
	roots := make([]nodeNumber, 0, len(nodes))

	for mInt := range nodes {
		if _, hasBefore := edges.index[1][mInt]; !hasBefore {
			roots = append(roots, mInt)
		}
	}

	for len(roots) > 0 {
		n := roots[0]
		roots = roots[1:]

		output = append(output, nodes[n])

		slc := make([]nodeNumber, 0, len(edges.index[0][n]))
		for m := range edges.index[0][n] {
			slc = append(slc, m)
		}

		sort.SliceStable(slc, func(i, j int) bool { return slc[i] < slc[j] }) // stabilize the output because we are using a map

		for _, m := range slc {
			e := edges.index[0][n][m]

			edges.removeIndex(edges.slice[e])

			if _, hasBefore := edges.index[1][m]; !hasBefore {
				roots = append(roots, m)
			}
		}
	}

	for pos := range edges.index {
		if len(edges.index[pos]) > 0 {
			cycle = make([]N, 0, len(edges.index[0]))
			for n := range edges.index[pos] {
				cycle = append(cycle, nodes[n])
			}

			return nil, cycle
		}
	}

	return output, nil
}
