package ipfscluster

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var Default_Port int = 9094

type IPFSClusterConnector struct {
	url        string
	selfID     string
	peerIDs    []string
	currentIdx int
}

// CreateIPFSClusterConnector is the constructor of IPFSClusterConnector
func CreateIPFSClusterConnector(port int) (*IPFSClusterConnector, error) {
	if port == 0 {
		port = Default_Port
	}
	conn := IPFSClusterConnector{url: fmt.Sprintf("http://127.0.0.1:%d", port)}
	_, err := conn.PeerInfo()
	if err != nil {
		return nil, err
	}
	_, err = conn.PeerLs()
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// PeerInfo list the info about the cluster peers
func (c *IPFSClusterConnector) PeerInfo() (string, error) {
	/* Return the connected peer info
	For the moment, only returns the name of the connected peer */
	resp, err := http.Get(c.url + "/id")
	if err != nil {
		return "", err
	}

	decoder := json.NewDecoder(resp.Body)
	var info map[string]interface{}
	if err = decoder.Decode(&info); err != nil {
		panic(err)
	}

	c.selfID = info["id"].(string)
	return info["peername"].(string), nil
}

// PeerLs list the number of peers that are inside the cluster
func (c *IPFSClusterConnector) PeerLs() (int, error) {
	/* List all peers inside the IPFS cluster
	For the moment, only returns the number of peers */
	resp, err := http.Get(c.url + "/peers")
	if err != nil {
		return 0, err
	}

	var peersInfo []map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var info map[string]interface{}
		if err = decoder.Decode(&info); err != nil {
			panic(err)
		}
		peersInfo = append(peersInfo, info)
		if info["id"].(string) != c.selfID {
			c.peerIDs = append(c.peerIDs, info["id"].(string))
		}
	}

	return len(peersInfo), nil
}

// PinStatus check the status of the specified cid, if the CID is not given, it will
// show all CIDs that are inside the ipfs cluster
func (c *IPFSClusterConnector) PinStatus(cid string) (string, error) {
	/* Check the pin status of all CIDs or a specific CID
	For the moment, only checks the number of pin peers */
	var statusURL string
	var pinStatus string
	if cid == "" {
		statusURL = c.url + "/pins"
	} else {
		statusURL = c.url + "/pins/" + cid
	}

	resp, err := http.Get(statusURL)
	if err != nil {
		return "", err
	}

	var pinInfo []map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var status map[string]interface{}
		if err = decoder.Decode(&status); err != nil {
			panic(err)
		}
		pinInfo = append(pinInfo, status)
	}

	for _, status := range pinInfo {
		var statusMap = status["peer_map"].(map[string]interface{})
		var pinCount int
		for key := range statusMap {
			if statusMap[key].(map[string]interface{})["status"].(string) == "pinned" {
				pinCount++
			}
		}
		pinStatus += fmt.Sprintf("%s pinned by %d peers.\n", status["cid"].(string), pinCount)
	}
	pinStatus = fmt.Sprintf("\nTotal number of pins: %d\n", len(pinInfo)) + pinStatus

	return pinStatus, err
}

// AddPin add the specified CID to the ipfs cluster, with the specified replication factor,
// the default behavior is recursive, which means pinning all content that is beneath the CID
// "mode" can be "direct" or "recursive"
func (c *IPFSClusterConnector) AddPin(cid string, replicationFactor int) error {
	/* Add a new CID to the cluster,  it uses the default replication
	factor that is specified in the CLUSTER configuration file */
	peerID := c.peerIDs[c.currentIdx]
	c.currentIdx = (c.currentIdx + 1) % len(c.peerIDs)
	postURL := fmt.Sprintf("%s/pins/ipfs/%s?mode=recursive&name=&replication-max=%d&replication-min=%d&shard-size=0&user-allocations=%s",
		c.url, cid, replicationFactor, replicationFactor, peerID)
	_, err := http.PostForm(postURL, nil)
	return err
}

// PeerLoad checks the load balance of the cluster, namely how many blocks is stored on each
// cluster peer
func (c *IPFSClusterConnector) PeerLoad() (string, error) {
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	max := func(a, b int) int {
		if a > b {
			return a
		}
		return b
	}

	statusURL := c.url + "/pins"
	resp, err := http.Get(statusURL)
	if err != nil {
		return "", err
	}

	var totBlocks int
	peerInfo := make(map[string]int)
	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var status map[string]interface{}
		if err = decoder.Decode(&status); err != nil {
			panic(err)
		}
		var statusMap = status["peer_map"].(map[string]interface{})
		for key := range statusMap {
			pinStatus := statusMap[key].(map[string]interface{})["status"].(string)
			if pinStatus == "pinned" {
				peerInfo[key]++
				totBlocks++
			}
		}
	}

	var peerLoad string
	minBlocks := totBlocks
	maxBlocks := 0
	var blocks []string
	for key := range peerInfo {
		minBlocks = min(minBlocks, peerInfo[key])
		maxBlocks = max(maxBlocks, peerInfo[key])
		blocks = append(blocks, strconv.Itoa(peerInfo[key]))
	}

	peerLoad += fmt.Sprintf("\nTotal blocks in the cluster: %d\n", totBlocks)
	peerLoad += fmt.Sprintf("Min blocks: %d, Max blocks: %d\n", minBlocks, maxBlocks)
	peerLoad += fmt.Sprintf("Detailed blocks info: %s", strings.Join(blocks, ", "))

	return peerLoad, nil
}
