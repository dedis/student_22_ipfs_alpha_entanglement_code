package test

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"ipfs-alpha-entanglement-code/entangler"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"log"
	"os"
	"sort"
	"strings"
	"testing"
)

var blockgetterTest = func(filepath string) func(*testing.T) {
	return func(t *testing.T) {
		alpha, s, p := 3, 5, 5
		c, err := ipfsconnector.CreateIPFSConnector(0)
		if err != nil {
			t.Fatal(err)
		}
		// add original file to ipfs
		cid, err := c.AddFile(filepath)
		if err != nil {
			t.Fatal(err)
		}

		// get merkle tree from IPFS and flatten the tree
		root, err := c.GetMerkleTree(cid, &entangler.Lattice{})
		if err != nil {
			t.Fatal(err)
		}

		nodesSwapped := root.GetFlattenedTree(s, p, true)

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
			outputPaths[k] = fmt.Sprintf("%s_entanglement_%d", strings.Split(filepath, ".")[0], k)
			defer os.Remove(outputPaths[k])
		}
		parityChan := make(chan entangler.EntangledBlock, alpha*len(nodesSwapped))
		err = tangler.Entangle(data, parityChan)
		if err != nil {
			t.Fatal(err)
		}
		err = tangler.WriteEntanglementToFile(maxSize, outputPaths, parityChan)
		if err != nil {
			t.Fatal(err)
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
			actualData, err := getter.GetData(i)
			if err != nil {
				t.Fatal(err)
			}
			expectedData, _ := nodesSwapped[i].Data()
			require.Equal(t, expectedData, actualData)
		}

		for i := 0; i < alpha; i++ {
			for j := 0; j < root.TreeSize; j++ {
				actualData, err := getter.GetParity(j+1, i)
				if err != nil {
					t.Fatal(err)
				}
				expectedData, _ := parityLeafNodes[i][j].Data()
				require.Equal(t, expectedData, actualData)
			}
		}
	}
}

func Test_Blockgetter_Basic(t *testing.T) {
	files, err := os.ReadDir("data")
	if err != nil {
		log.Fatal(err)
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		t.Run("basic_"+fileName, blockgetterTest("data/"+fileName))
	}
}
