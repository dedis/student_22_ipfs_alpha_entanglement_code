package test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"strings"
	"testing"
)

var blockgetterTest = func() func(*testing.T) {
	return func(t *testing.T) {
		alpha, s, p := 3, 5, 5
		path := "data/largeFile.txt"
		c, err := ipfsconnector.CreateIPFSConnector(0)

		// add original file to ipfs
		cid, err := c.AddFile(path)

		// get merkle tree from IPFS and flatten the tree
		root, err := c.GetMerkleTree(cid, &entangler.Lattice{})
		if err != nil {
			t.Fail()
		}
		nodesNotSwapped := root.GetFlattenedTree(s, p, false)
		nodesSwapped := root.GetFlattenedTree(s, p, true)

		var dataCIDs []string
		for _, node := range nodesNotSwapped {
			dataCIDs = append(dataCIDs, node.CID)
		}

		// generate entanglement
		data := make(chan []byte, len(nodesSwapped))
		maxSize := 0
		for _, node := range nodesSwapped {
			nodeData, err := node.Data()
			if err != nil {
				t.Fail()
			}
			data <- nodeData
			if len(nodeData) > maxSize {
				maxSize = len(nodeData)
			}
		}
		close(data)
		tangler := entangler.NewEntangler(alpha, s, p)

		outputPaths := make([]string, alpha)
		for k := 0; k < alpha; k++ {
			outputPaths[k] = fmt.Sprintf("%s_entanglement_%d", strings.Split(path, ".")[0], k)
		}
		err = tangler.Entangle(data)
		if err != nil {
			t.Fail()
		}
		err = tangler.WriteEntanglementToFile(maxSize, outputPaths)
		if err != nil {
			t.Fail()
		}

		// upload entanglements to ipfs
		var parityCIDs [][]string
		var parityLeafNodes [][]*ipfsconnector.TreeNode
		for _, entanglementFilename := range outputPaths {
			cid, err := c.AddFile(entanglementFilename)
			// get merkle tree from IPFS and flatten the tree
			root, err := c.GetMerkleTree(cid, &entangler.Lattice{})
			if err != nil {
				t.Fail()
			}
			nodesLeaf := root.GetLeafNodes()

			var singleParityCIDs []string
			for _, node := range nodesLeaf {
				singleParityCIDs = append(singleParityCIDs, node.CID)
			}
			parityCIDs = append(parityCIDs, singleParityCIDs)
			parityLeafNodes = append(parityLeafNodes, nodesLeaf)
		}

		// Create getter
		getter := ipfsconnector.CreateIPFSGetter(c, dataCIDs, parityCIDs, nodesSwapped)

		// Verify that we get the expected results
		for i := 0; i < root.TreeSize; i++ {
			actualData, err := getter.GetData(i + 1)
			if err != nil {
				t.Fail()
			}
			expectedData, _ := nodesSwapped[i].Data()
			require.Equal(t, expectedData, actualData)
		}

		for i := 0; i < alpha; i++ {
			for j := 0; j < root.TreeSize; j++ {
				actualData, err := getter.GetParity(j+1, i)
				if err != nil {
					t.Fail()
				}
				expectedData, _ := parityLeafNodes[i][j].Data()
				require.Equal(t, expectedData, actualData)
			}
		}
	}
}

func Test_Blockgetter_Basic(t *testing.T) {
	t.Run("basic", blockgetterTest())
}
