package ipfsconnector

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	icore "github.com/ipfs/interface-go-ipfs-core"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/kubo/config"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"
)

// IPFSConnector manages all the interaction with IPFS node
type IPFSConnector struct {
	icore.CoreAPI
	node *core.IpfsNode

	ctx      context.Context
	stopFunc context.CancelFunc
}

var loadPluginsOnce sync.Once

// CreateIPFSConnector creates a running IPFS node and returns a connector to it
func CreateIPFSConnector(enableExperimentalFeature bool) (*IPFSConnector, error) {
	// setup plugins
	var onceErr error
	loadPluginsOnce.Do(func() {
		onceErr = SetupPlugins("")
	})
	if onceErr != nil {
		return nil, onceErr
	}

	// create IPFS repo
	repoPath, err := CreateIPFSRepo(enableExperimentalFeature)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// create IPFS node
	ctx, cancel := context.WithCancel(context.Background())
	node, err := CreateNode(ctx, repoPath)
	if err != nil {
		cancel()
		return nil, err
	}
	api, err := coreapi.NewCoreAPI(node)
	if err != nil {
		cancel()
		return nil, err
	}

	return &IPFSConnector{
		CoreAPI:  api,
		node:     node,
		ctx:      ctx,
		stopFunc: cancel,
	}, nil
}

// Stop stops a running IPFS node
func (c *IPFSConnector) Stop() {
	c.stopFunc()
}

// ConnectToPeers connects the current node to peers in the given list
func (c *IPFSConnector) ConnectToPeers(peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := multiaddr.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := c.Swarm().Connect(c.ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

// SetupPlugins prepares and sets up the plugins
func SetupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}
	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

// CreateIPFSRepo creates an IPFS repo
func CreateIPFSRepo(enableExperimentalFeature bool) (string, error) {
	repoPath, err := os.MkdirTemp("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(io.Discard, 2048)
	if err != nil {
		return "", err
	}

	// When creating the repository, you can define custom settings on the repository, such as enabling experimental
	// features (See experimental-features.md) or customizing the gateway endpoint.
	// To do such things, you should modify the variable `cfg`. For example:
	if enableExperimentalFeature {
		// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-filestore
		cfg.Experimental.FilestoreEnabled = true
		// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-urlstore
		cfg.Experimental.UrlstoreEnabled = true
		// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#ipfs-p2p
		cfg.Experimental.Libp2pStreamMounting = true
		// https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md#p2p-http-proxy
		cfg.Experimental.P2pHttpProxy = true
		// See also: https://github.com/ipfs/kubo/blob/master/docs/config.md
		// And: https://github.com/ipfs/kubo/blob/master/docs/experimental-features.md
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

// CreateNode creates a new IPFS node and starts running it
func CreateNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	return core.NewNode(ctx, nodeOptions)
}
