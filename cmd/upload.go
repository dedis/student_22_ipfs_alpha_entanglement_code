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
	err = c.InitIPFSConnector()
	if err != nil {
		return "", "", nil, err
	}

	// add original file to ipfs
	rootCID, err = c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", rootCID, path)

	if alpha < 1 {
		// expect no entanglement
		return rootCID, "", nil, nil
	}

	clusterErr := c.InitIPFSClusterConnector()

	// get merkle tree from IPFS and flatten the tree
	root, err := c.GetMerkleTree(rootCID, &entangler.Lattice{})
	if err != nil {
		return rootCID, "", nil, xerrors.Errorf("could not read merkle tree: %s", err)
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
	parityPinResult := make([][]bool, alpha)
	for k := 0; k < alpha; k++ {
		parityCIDs[k] = make([]string, blockNum)
		parityPinResult[k] = make([]bool, blockNum)
	}

	var waitGroupUpload sync.WaitGroup
	var waitGroupPin sync.WaitGroup
	for parityBlock := range parityChan {
		waitGroupUpload.Add(1)
		waitGroupPin.Add(1)

		go func(block entangler.EntangledBlock) {
			defer waitGroupUpload.Done()

			// upload file to IPFS network
			blockCID, err := c.AddFileFromMem(block.Data)
			if err == nil {
				parityCIDs[block.Strand][block.LeftBlockIndex-1] = blockCID
			}

			// pin file in cluster
			if clusterErr == nil {
				go func() {
					defer waitGroupPin.Done()

					err := c.AddPin(blockCID, 1)
					if err == nil {
						parityPinResult[block.Strand][block.LeftBlockIndex-1] = true
					}
				}()
			}
		}(parityBlock)
	}
	waitGroupUpload.Wait()

	// check if all parity blocks are added successfully
	for k := 0; k < alpha; k++ {
		for i, parity := range parityCIDs[k] {
			if len(parity) == 0 {
				return rootCID, "", nil, xerrors.Errorf("could not upload parity %d on strand %d\n", i, k)
			}
		}
		util.LogPrint("Finish uploading entanglement %d", k)
	}

	// Store Metatdata
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

	util.LogPrint("File CID: %s. MetaFile CID: %s", rootCID, metaCID)
	if clusterErr != nil {
		return rootCID, metaCID, nil, clusterErr
	}

	pinResult = func() (err error) {
		err = c.AddPin(metaCID, 0)
		if err != nil {
			return xerrors.Errorf("could not pin metadata: %s", err)
		}

		waitGroupPin.Wait()
		// check if all parity blocks are pinned successfully
		for k := 0; k < alpha; k++ {
			for i, success := range parityPinResult[k] {
				if !success {
					return xerrors.Errorf("could not pin parity %d on strand %d\n", i, k)
				}
			}
			util.LogPrint("Finish pinning entanglement %d", k)
		}

		return nil
	}

	return rootCID, metaCID, pinResult, nil
}
