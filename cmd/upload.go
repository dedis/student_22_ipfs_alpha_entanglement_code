package cmd

import (
	"encoding/json"
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"sync"

	"golang.org/x/xerrors"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int) (rootCID string, metaCID string, err error) {
	err = c.InitIPFSConnector()
	if err != nil {
		return "", "", err
	}

	// add original file to ipfs
	rootCID, err = c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", rootCID, path)

	if alpha < 1 {
		// expect no entanglement
		return rootCID, "", nil
	}

	err = c.InitIPFSClusterConnector()
	if err != nil {
		return rootCID, "", err
	}

	// get merkle tree from IPFS and flatten the tree
	root, err := c.GetMerkleTree(rootCID, &entangler.Lattice{})
	if err != nil {
		return rootCID, "", xerrors.Errorf("could not read merkle tree: %s", err)
	}
	nodes := root.GetFlattenedTree(s, p, true)
	blockNum := len(nodes)
	util.InfoPrint(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), blockNum)
	for _, node := range nodes {
		util.InfoPrint(util.Green(" %d"), node.PreOrderIdx)
	}
	util.InfoPrint("\n")
	util.LogPrint("Finish reading and flattening file's merkle tree from IPFS")

	// generate entanglement
	dataChan := make(chan []byte, blockNum)
	parityChan := make(chan entangler.EntangledBlock, alpha*blockNum)

	tangler := entangler.NewEntangler(alpha, s, p)
	go func() {
		err = tangler.Entangle(dataChan, parityChan)
		if err != nil {
			panic(xerrors.Errorf("could not generate entanglement: %s", err))
		}
	}()

	// send data to entangler
	go func() {
		for _, node := range nodes {
			nodeData, err := node.Data()
			if err != nil {
				return
				// return rootCID, "", xerrors.Errorf("could not load chunk data from IPFS: %s", err)
			}
			dataChan <- nodeData
		}
		close(dataChan)
	}()

	// store parity blocks one by one
	parityCIDs := make([][]string, alpha)
	for k := 0; k < alpha; k++ {
		parityCIDs[k] = make([]string, blockNum)
	}

	var waitGroup sync.WaitGroup
	for parityBlock := range parityChan {
		waitGroup.Add(1)

		go func(block entangler.EntangledBlock) {
			defer waitGroup.Done()

			blockCID, err := c.AddAndPinAsFile(block.Data, 1)
			if err == nil {
				parityCIDs[block.Strand][block.LeftBlockIndex-1] = blockCID
			}
		}(parityBlock)
	}
	waitGroup.Wait()

	// check if all parity blocks are added and pinned successfully
	for k := 0; k < alpha; k++ {
		for i, parity := range parityCIDs[k] {
			if len(parity) == 0 {
				return rootCID, "", xerrors.Errorf("could not upload parity %d on strand %d\n", i, k)
			}
		}
		util.LogPrint("Finish uploading and pinning entanglement %d", k)
	}

	// Store Metatdata?
	cidMap := make(map[string]int)
	for i, node := range nodes {
		cidMap[node.CID] = i + 1
	}
	metaData := Metadata{
		Alpha:           alpha,
		S:               s,
		P:               p,
		RootCID:         rootCID,
		DataCIDIndexMap: cidMap,
		ParityCIDs:      parityCIDs,
	}
	rawMetadata, err := json.Marshal(metaData)
	if err != nil {
		return rootCID, "", xerrors.Errorf("could not marshal metadata: %s", err)
	}
	metaCID, err = c.AddAndPinAsFile(rawMetadata, 0)
	if err != nil {
		return rootCID, "", xerrors.Errorf("could not upload metadata: %s", err)
	}

	return rootCID, metaCID, nil
}
