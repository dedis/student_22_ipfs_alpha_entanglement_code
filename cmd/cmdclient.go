package cmd

import "sync"

type Metadata struct {
	Alpha int
	S     int
	P     int

	CIDIndexMap map[string]int
}

type Client struct {
	*sync.RWMutex
	Metadata map[string]*Metadata
}

func NewClient() (client *Client) {
	return &Client{Metadata: make(map[string]*Metadata)}
}

func (c *Client) AddMetaData(rootCID string, metadata *Metadata) {
	c.Lock()
	defer c.Unlock()

	c.Metadata[rootCID] = metadata
}

func (c *Client) GetMetaData(rootCID string) (metadata *Metadata, ok bool) {
	c.RLock()
	defer c.Unlock()

	metadata, ok = c.Metadata[rootCID]
	return
}
