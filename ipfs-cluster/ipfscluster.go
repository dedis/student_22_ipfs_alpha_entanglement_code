package ipfscluster

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var Default_Port int = 9094

type IPFSClusterConnector struct {
	url string
}

// CreateIPFSClusterConnector is the constructor of IPFSClusterConnector
func CreateIPFSClusterConnector(port int) (*IPFSClusterConnector, error) {
	if port == 0 {
		port = Default_Port
	}
	return &IPFSClusterConnector{fmt.Sprintf("http://127.0.0.1:%d", port)}, nil
}

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

	return info["peername"].(string), nil
}

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
	}

	return len(peersInfo), nil
}

func (c *IPFSClusterConnector) PinStatus(cid string) (string, error) {
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
		pinStatus += fmt.Sprintf("\n%s pinned by %d peers.", status["cid"].(string),
			len(status["peer_map"].(map[string]interface{})))
	}

	return pinStatus, err
}
