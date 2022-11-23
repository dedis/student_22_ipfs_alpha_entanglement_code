package cmd

import (
	"encoding/json"
	ipfscluster "ipfs-alpha-entanglement-code/ipfs-cluster"
	ipfsconnector "ipfs-alpha-entanglement-code/ipfs-connector"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"
)

type Metadata struct {
	Alpha int
	S     int
	P     int

	RootCID string

	DataCIDIndexMap map[string]int
	ParityCIDs      [][]string
}

type Client struct {
	*ipfsconnector.IPFSConnector
	*ipfscluster.IPFSClusterConnector

	*cobra.Command
}

// NewClient creates a new client for futhur use
func NewClient() (client *Client, err error) {
	conn, err := ipfsconnector.CreateIPFSConnector(0)
	if err != nil {
		return nil, xerrors.Errorf("fail to connect to IPFS: %s", err)
	}

	conn2, err := ipfscluster.CreateIPFSClusterConnector(0)
	if err != nil {
		return nil, xerrors.Errorf("fail to connect to cluster: %s", err)
	}

	client = &Client{
		IPFSConnector:        conn,
		IPFSClusterConnector: conn2,
	}
	client.initCmd()

	return client, nil
}

// AddAndPinAsFile adds a file to IPFS network and pin the file in cluster with a replication factor
// replicate = 0 means use default config in the cluster
// TODO: add replicate config in pinning
func (c *Client) AddAndPinAsFile(data []byte, replicate int) (cid string, err error) {
	// write the data to a temp file
	tempfile, err := os.CreateTemp("", "IPFS-file-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tempfile.Name())
	tempfile.Write(data)

	// upload file to IPFS network
	cid, err = c.AddFile(tempfile.Name())
	if err != nil {
		return "", err
	}

	// pin file in cluster
	err = c.AddPin(cid)
	return cid, err
}

// AddAndPinAsRaw adds raw data to IPFS network and pin it in cluster with a replication factor
// replicate = 0 means use default config in the cluster
// TODO: add replicate config in pinning
func (c *Client) AddAndPinAsRaw(data []byte, replicate int) (cid string, err error) {
	// upload raw bytes to IPFS network
	cid, err = c.AddRawData(data)
	if err != nil {
		return "", err
	}

	// pin data in cluster
	err = c.AddPin(cid)
	return cid, err
}

// GetMetaData downloads metafile from IPFS network and returns a metafile object
func (c *Client) GetMetaData(cid string) (metadata *Metadata, err error) {
	// create temp file to store metadata
	tempfile, err := os.CreateTemp("", "IPFS-file-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tempfile.Name())

	// download metadata from IPFS network
	err = c.GetFile(cid, tempfile.Name())
	if err != nil {
		return nil, err
	}

	// unmarshal to Metadata object
	data, err := os.ReadFile(tempfile.Name())
	if err != nil {
		return nil, err
	}
	var myMetadata Metadata
	err = json.Unmarshal(data, &myMetadata)
	if err != nil {
		return nil, err
	}

	return &myMetadata, nil
}
