package ipfscluster

import (
	"fmt"
	"io"
	"net/http"
)

type IPFSClusterConnector struct {
	url string
}

var Default_Port int = 9094

// CreateIPFSClusterConnector is the constructor of IPFSClusterConnector
func CreateIPFSClusterConnector(port int) (*IPFSClusterConnector, error) {
	if port == 0 {
		port = Default_Port
	}
	return &IPFSClusterConnector{fmt.Sprintf("http://127.0.0.1:%d", port)}, nil
}

func (c *IPFSClusterConnector) PeerLs() (string, error) {
	resp, err := http.Get(c.url + "/peers")
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (c *IPFSClusterConnector) PeerInfo() (string, error) {
	resp, err := http.Get(c.url + "/id")
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
