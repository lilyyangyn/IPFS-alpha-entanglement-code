package ipfsconnector

// TreeNode implements a node in IPLD Merkle Tree
type TreeNode struct {
	// TODO: consider DAG? Multiple Parents?
	Data         []byte
	Children     []*TreeNode
	Parent       *TreeNode
	Depth        int
	TreeSize     int
	PostOrderIdx int
}

// CreateTreeNode is the constructor of TreeNode
func CreateTreeNode(data []byte) *TreeNode {
	n := TreeNode{Data: data, Parent: nil}
	n.Children = make([]*TreeNode, 0)
	n.TreeSize = 1
	n.Depth = -1
	return &n
}

// AddChild links a child to the current node
func (n *TreeNode) AddChild(child *TreeNode) {
	n.Children = append(n.Children, child)
	n.TreeSize += child.TreeSize
	child.Parent = n
	child.Depth = n.Depth + 1
}

// GetFlattenedTree removes dependencies inside lattice windows and returns an array of tree nodes
func (n *TreeNode) GetFlattenedTree(s int, p int) []*TreeNode {
	nodes := make([]*TreeNode, n.TreeSize)

	// post order traversal of the tree
	internals := make([]*TreeNode, 0)
	var walker func(*TreeNode)
	walker = func(parent *TreeNode) {
		if parent == nil {
			return
		}
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				walker(child)
			}
			// TODO: should root be inlcuded?
			internals = append(internals, parent)
		}
		nodes[parent.PostOrderIdx] = parent
	}
	walker(n)

	// move the parents at least one LW away from their children
	windowSize := s * p
	for _, internalNode := range internals {
		lowestChild := internalNode.Children[0]
		highestChild := internalNode.Children[len(internalNode.Children)-1]
		for j := windowSize; j < n.TreeSize; j += windowSize + s {
			inWindow := (nodes[j].PostOrderIdx > lowestChild.PostOrderIdx-windowSize &&
				nodes[j].PostOrderIdx < highestChild.PostOrderIdx+windowSize)
			if !inWindow && len(nodes[j].Children) == 0 {
				// Swap position of internalNode and the data
				nodes[j], nodes[internalNode.PostOrderIdx-1] = nodes[internalNode.PostOrderIdx-1], nodes[j]
				break
			}
		}
	}
	return nodes
}