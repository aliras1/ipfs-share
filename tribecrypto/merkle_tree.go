// Copyright (c) 2019 Laszlo Sari
//
// FileTribe is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// FileTribe is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
//

package tribecrypto


import (
	"bytes"

	
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)


type node struct {
	value   []byte
	parent  *node
	sibling *node
	lchild  *node
	rchild  *node
}

type MerkleTree struct {
	root   *node
	leaves []*node
	depth  int
}

func NewMerkleTree(leaves [][]byte) *MerkleTree {
	tree := &MerkleTree{}

	var unprocessed []*node
	for i := 0; i < len(leaves); i++ {
		n := &node{
			value: leaves[i],
		}
		
		unprocessed = append(unprocessed, n)
	}

	for len(unprocessed) > 1 {
		lnode := unprocessed[0]
		rnode := unprocessed[1]

		if lnode.isLeaf() {
			tree.leaves = append(tree.leaves, lnode)
		}
		if rnode.isLeaf() {
			tree.leaves = append(tree.leaves, rnode)
		}

		lnode.sibling = rnode
		rnode.sibling = lnode

		var value []byte
		if bytes.Compare(lnode.value, rnode.value) < 0 {
			value = ethcrypto.Keccak256(lnode.value, rnode.value)
		} else {
			value = ethcrypto.Keccak256(rnode.value, lnode.value)
		}
		newNode := &node{
			value:  value,
			lchild: lnode,
			rchild: rnode,
		}
		lnode.parent = newNode
		rnode.parent = newNode

		unprocessed = append(unprocessed[2:], newNode)
	}

	tree.root = unprocessed[0]

	return tree
}

func (tree *MerkleTree) Prove(leafValue []byte) ([][]byte, error) {
	var n *node
	for _, leaf := range tree.leaves {
		if bytes.EqualFold(leafValue, leaf.value) {
			n = leaf
			break
		}
	}

	if n == nil {
		return [][]byte{}, errors.New("leafValue not found")
	}

	var proof [][]byte
	for n.parent != nil {
		proof = append(proof, n.sibling.value)
		n = n.parent
	}

	return proof, nil
}

func (tree *MerkleTree) Root() []byte {
	return tree.root.value
}

func (tree *MerkleTree) Leaves() [][]byte {
	leaveValues := make([][]byte, len(tree.leaves))
	for i := 0; i < len(tree.leaves); i++ {
		leaveValues[i] = tree.leaves[i].value
	}
	return leaveValues
}


func VerifyMerkleProof(proof [][]byte, root []byte, leaf []byte) bool {
	computedHash := leaf

	for i := 0; i < len(proof); i++ {
		proofElement := proof[i]

		if bytes.Compare(computedHash, proofElement) < 0 {
			// Hash(current computed hash + current element of the proof)
			computedHash = ethcrypto.Keccak256(computedHash, proofElement)
		} else {
			// Hash(current element of the proof + current computed hash)
			computedHash = ethcrypto.Keccak256(proofElement, computedHash)
		}
	}

	// Check if the computed hash (root) is equal to the provided root
	return bytes.EqualFold(computedHash, root)
}

func (n *node) isLeaf() bool {
	if n.lchild == nil && n.rchild == nil {
		return true
	}
	return false
}