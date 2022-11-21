package cmd

import "sync"

type Metadata struct {
	Alpha int
	S     int
	P     int

	DataCIDIndexMap map[string]int
	ParityCIDs      [][]string
	DataFilter      map[int]struct{}
}

type Client struct {
	*sync.RWMutex
	Metadata map[string]*Metadata
}

func NewClient() (client *Client) {
	return &Client{RWMutex: &sync.RWMutex{}, Metadata: make(map[string]*Metadata)}
}

func (c *Client) AddMetaData(rootCID string, metadata *Metadata) {
	c.Lock()
	defer c.Unlock()

	c.Metadata[rootCID] = metadata
}

func (c *Client) GetMetaData(rootCID string) (metadata *Metadata, ok bool) {
	c.RLock()
	defer c.RUnlock()

	metadata, ok = c.Metadata[rootCID]
	return
}
