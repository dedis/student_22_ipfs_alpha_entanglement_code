package test

import (
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"os"
	"strings"
	"testing"
)

var blockgetterTest = func() func(*testing.T) {
	return func(t *testing.T) {
		alpha, s, p := 3, 5, 5
		path := "data/largeFile.txt"
		c, err := ipfsconnector.CreateIPFSConnector(0)
		if err != nil {
			t.Fatal(err)
		}
		// add original file to ipfs
		cid, err := c.AddFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// get merkle tree from IPFS and flatten the tree
		root, err := c.GetMerkleTree(cid, &entangler.Lattice{})
		if err != nil {
			t.Fatal(err)
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
				t.Fatal(err)
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
		parityChan := make(chan entangler.EntangledBlock, alpha*len(nodesSwapped))
		err = tangler.Entangle(data, parityChan)
		if err != nil {
			t.Fatal(err)
		}

		parities := make([][][]byte, alpha)
		for k := 0; k < alpha; k++ {
			parities[k] = make([][]byte, len(nodesSwapped))
		}
		for parity := range parityChan {
			c := make([]byte, maxSize)
			copy(c, parity.Data)
			parities[parity.Strand][parity.LeftBlockIndex-1] = c
		}

		for k := 0; k < alpha; k++ {
			// generate byte array of the current strand
			entangledData := make([]byte, 0)
			for _, parityData := range parities[k] {
				entangledData = append(entangledData, parityData...)
			}

			// write entanglement to file
			err = os.WriteFile(outputPaths[k], entangledData, 0644)
			if err != nil {
				t.Fatal(err)
				return
			}
		}

		// upload entanglements to ipfs
		var parityCIDs [][]string
		var parityLeafNodes [][]*ipfsconnector.TreeNode
		for _, entanglementFilename := range outputPaths {
			cid, err := c.AddFile(entanglementFilename)
			if err != nil {
				t.Fatal(err)
			}
			// get merkle tree from IPFS and flatten the tree
			root, err := c.GetMerkleTree(cid, &entangler.Lattice{})
			if err != nil {
				t.Fatal(err)
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
		cidMap := make(map[string]int)
		for i, node := range nodesSwapped {
			cidMap[node.CID] = i
		}
		getter := ipfsconnector.CreateIPFSGetter(c, cidMap, parityCIDs)

		// Verify that we get the expected results
		for i := 0; i < root.TreeSize; i++ {
			actualData, err := getter.GetData(i + 1)
			if err != nil {
				t.Fatal(err)
			}
			expectedData, _ := nodesSwapped[i].Data()
			// require.Equal(t, expectedData, actualData)
			fmt.Println(len(actualData), len(expectedData))
		}

		for i := 0; i < alpha; i++ {
			for j := 0; j < root.TreeSize; j++ {
				actualData, err := getter.GetParity(j+1, i)
				if err != nil {
					t.Fatal(err)
				}
				expectedData, _ := parityLeafNodes[i][j].Data()
				// require.Equal(t, expectedData, actualData)
				fmt.Println(len(actualData), len(expectedData))
			}
		}
	}
}

func Test_Blockgetter_Basic(t *testing.T) {
	t.Run("basic", blockgetterTest())
}
