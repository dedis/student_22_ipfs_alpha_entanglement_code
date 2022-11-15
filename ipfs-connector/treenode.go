package ipfsconnector

// TreeNode implements a node in IPLD Merkle Tree
type TreeNode struct {
	// TODO: consider DAG? Multiple Parents?
	data []byte

	Children    []*TreeNode
	Parent      *TreeNode
	Depth       int
	TreeSize    int
	LeafSize    int
	PreOrderIdx int

	connector *IPFSConnector
	CID       string
}

// CreateTreeNode is the constructor of TreeNode
func CreateTreeNode(data []byte) *TreeNode {
	n := TreeNode{data: data, Parent: nil}
	n.Children = make([]*TreeNode, 0)
	n.TreeSize = 1
	n.LeafSize = 0
	n.Depth = -1
	return &n
}

// LoadData loads the node raw data from IPFS network lazily
func (n *TreeNode) Data() (data []byte, err error) {
	if len(n.data) == 0 && n.connector != nil && len(n.CID) > 0 {
		var myData []byte
		myData, err = n.connector.shell.BlockGet(n.CID)
		if err != nil {
			return
		}
		n.data = myData
	}
	data = n.data

	return
}

// AddChild links a child to the current node
func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
	n.TreeSize += child.TreeSize
	n.LeafSize += child.LeafSize
	child.Parent = n
	child.Depth = n.Depth + 1
}

// GetFlattenedTree removes dependencies inside lattice windows and returns an array of tree nodes
func (n *TreeNode) GetFlattenedTree(s int, p int, swap bool) []*TreeNode {
	nodes := make([]*TreeNode, n.TreeSize)

	// preorder traversal of the tree
	internals := make([]*TreeNode, 0)
	var walker func(*TreeNode)
	walker = func(parent *TreeNode) {
		if parent == nil {
			return
		}
		nodes[parent.PreOrderIdx] = parent
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
			// meaningless to include root
			if parent != n {
				internals = append(internals, parent)
			}
		}
	}
	walker(n)

	if swap {
		// move the parents at least one LW away from their children
		windowSize := s * p
		for _, internalNode := range internals {
			lowestChild := internalNode.Children[0]
			highestChild := internalNode.Children[len(internalNode.Children)-1]
			for j := windowSize; j < n.TreeSize; j += s {
				inWindow := (nodes[j].PreOrderIdx > lowestChild.PreOrderIdx-windowSize &&
					nodes[j].PreOrderIdx < highestChild.PreOrderIdx+windowSize)
				if !inWindow && len(nodes[j].Children) == 0 {
					// Swap position of internalNode and the data
					nodes[j], nodes[internalNode.PreOrderIdx-1] = nodes[internalNode.PreOrderIdx-1], nodes[j]
					break
				}
			}
		}
	}
	return nodes
}

// GetLeafNodes returns all tree leaves
func (n *TreeNode) GetLeafNodes() []*TreeNode {
	nodes := make([]*TreeNode, n.LeafSize)

	// preorder traversal of the tree
	var walker func(*TreeNode)
	var leafCnt = 0
	walker = func(parent *TreeNode) {
		if parent == nil {
			return
		}
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
		} else {
			nodes[leafCnt] = parent
			leafCnt++
		}
	}
	walker(n)
	return nodes
}
