package cmd

import (
	"encoding/json"
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"strings"

	"golang.org/x/xerrors"
)

// Upload uploads the original file, generates and uploads the entanglement of that file
func (c *Client) Upload(path string, alpha int, s int, p int) (rootCID string, metaCID string, err error) {
	// add original file to ipfs
	rootCID, err = c.AddFile(path)
	util.CheckError(err, "could not add File to IPFS")
	util.LogPrint("Finish adding file to IPFS with CID %s. File path: %s", rootCID, path)

	if alpha < 1 {
		// expect no entanglement
		return rootCID, "", nil
	}

	// get merkle tree from IPFS and flatten the tree
	root, err := c.GetMerkleTree(rootCID, &entangler.Lattice{})
	if err != nil {
		return rootCID, "", xerrors.Errorf("could not read merkle tree: %s", err)
	}
	nodes := root.GetFlattenedTree(s, p, true)
	util.InfoPrint(util.Green("Number of nodes in the merkle tree is %d. Node sequence:"), len(nodes))
	for _, node := range nodes {
		util.InfoPrint(util.Green(" %d"), node.PreOrderIdx)
	}
	util.InfoPrint("\n")
	util.LogPrint("Finish reading and flattening file's merkle tree from IPFS")

	// generate entanglement
	data := make(chan []byte, len(nodes))
	maxSize := 0
	for _, node := range nodes {
		nodeData, err := node.Data()
		if err != nil {
			return rootCID, "", xerrors.Errorf("could not load chunk data from IPFS: %s", err)
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
		return rootCID, "", xerrors.Errorf("could not generate entanglement: %s", err)
	}

	// store parity blocks one by one
	parityCIDs := make([][]string, alpha)
	for k, parityBlocks := range tangler.ParityBlocks {
		cids := make([]string, 0)
		for i, block := range parityBlocks {
			blockCID, err := c.AddAndPinAsRaw(block.Data, 0)
			if err != nil {
				return rootCID, "", xerrors.Errorf("could not upload entanglement %d on Strand %d: %s", i, k, err)
			}
			cids = append(cids, blockCID)
		}
		parityCIDs[k] = cids
		util.LogPrint("Finish generating entanglement %d", k)
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
