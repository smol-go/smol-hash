package consistenthash

// Node represents a physical node in the hash ring
type Node struct {
	ID       string            // Unique identifier for the node
	Host     string            // Host address (e.g., "192.168.1.1:8080")
	Load     int               // Current number of keys assigned to this node
	MaxLoad  int               // Maximum allowed load for bounded loads
	Metadata map[string]string // Additional metadata
}

// NewNode creates a new node with the given ID and host
func NewNode(id, host string) *Node {
	return &Node{
		ID:       id,
		Host:     host,
		Load:     0,
		MaxLoad:  0,
		Metadata: make(map[string]string),
	}
}

// CanAcceptKey checks if the node can accept more keys (for bounded loads)
func (n *Node) CanAcceptKey() bool {
	if n.MaxLoad == 0 {
		return true // No limit set
	}
	return n.Load < n.MaxLoad
}

// IncrementLoad increases the load counter
func (n *Node) IncrementLoad() {
	n.Load++
}

// DecrementLoad decreases the load counter
func (n *Node) DecrementLoad() {
	if n.Load > 0 {
		n.Load--
	}
}

// ResetLoad sets the load counter to zero
func (n *Node) ResetLoad() {
	n.Load = 0
}
