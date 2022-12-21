package cmd

import (
	"encoding/json"
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"sync"

	"golang.org/x/xerrors"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int) (rootCID string, metaCID string, pinResult func() error, err error) {
	// init ipfs connector. Fail the whole process if no connection built
	err = c.InitIPFSConnector()
	if err != nil {
		return "", "", nil, err
	}

	/* add original file to ipfs */

	rootCID, err = c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrintf("Finish adding file to IPFS with CID %s. File path: %s", rootCID, path)

	if alpha < 1 {
		// expect no entanglement
		return rootCID, "", nil, nil
	}

	// init cluster connector. Delay th fail after all uploading to IPFS finishes
	clusterErr := c.InitIPFSClusterConnector()

	/* get merkle tree from IPFS and flatten the tree */

	root, err := c.GetMerkleTree(rootCID, &entangler.Lattice{})
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not read merkle tree: %s", err)
	}
	nodes := root.GetFlattenedTree(s, p, true)
	blockNum := len(nodes)
	util.InfoPrintf(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), blockNum)
	for _, node := range nodes {
		util.InfoPrintf(util.Green(" %d"), node.PreOrderIdx)
	}
	util.InfoPrintf("\n")
	util.LogPrintf("Finish reading and flattening file's merkle tree from IPFS")

	/* generate entanglement */

	dataChan := make(chan []byte, blockNum)
	parityChan := make(chan entangler.EntangledBlock, alpha*blockNum)

	// start the entangler to read from pipline
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
			}
			dataChan <- nodeData
		}
		close(dataChan)
	}()

	/* store parity blocks one by one */

	parityCIDs := make([][]string, alpha)
	for k := 0; k < alpha; k++ {
		parityCIDs[k] = make([]string, blockNum)
	}

	var waitGroupAdd sync.WaitGroup
	for block := range parityChan {
		waitGroupAdd.Add(1)

		go func(block entangler.EntangledBlock) {
			defer waitGroupAdd.Done()

			// upload file to IPFS network
			blockCID, err := c.AddFileFromMem(block.Data)
			if err == nil {
				parityCIDs[block.Strand][block.LeftBlockIndex-1] = blockCID
			}
		}(block)
	}
	waitGroupAdd.Wait()

	// check if all parity blocks are added successfully
	for k := 0; k < alpha; k++ {
		for i, parity := range parityCIDs[k] {
			if len(parity) == 0 {
				return rootCID, "", nil, xerrors.Errorf("could not upload parity %d on strand %d\n", i, k)
			}
		}
		util.LogPrintf("Finish uploading entanglement %d", k)
	}

	/* Store Metatdata */

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
		return rootCID, "", nil, xerrors.Errorf("could not marshal metadata: %s", err)
	}
	metaCID, err = c.AddFileFromMem(rawMetadata)
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not upload metadata: %s", err)
	}

	util.LogPrintf("File CID: %s. MetaFile CID: %s", rootCID, metaCID)
	if clusterErr != nil {
		return rootCID, metaCID, nil, clusterErr
	}

	/* pin files in cluster */

	var waitGroupPin sync.WaitGroup
	waitGroupPin.Add(1)
	var PinErr error
	go func() {
		defer waitGroupPin.Done()

		err = c.AddPin(metaCID, 0)
		if err != nil {
			PinErr = xerrors.Errorf("could not pin metadata: %s", err)
			return
		}

		for i := 0; i < alpha; i++ {
			for j := 0; j < len(parityCIDs[0]); j++ {
				err := c.AddPin(parityCIDs[i][j], 1)
				if err != nil {
					PinErr = xerrors.Errorf("could not pin parity %s: %s", parityCIDs[i][j], err)
					return
				}
			}
		}
	}()

	pinResult = func() (err error) {
		waitGroupPin.Wait()
		return PinErr
	}

	return rootCID, metaCID, pinResult, nil
}
