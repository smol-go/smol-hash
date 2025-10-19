package consistenthash

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

// HashRing represents the consistent hash ring
type HashRing struct {
	mu            sync.RWMutex
	nodes         map[string]*Node  // nodeID -> Node
	ring          []uint32          // sorted hash values
	ringMap       map[uint32]string // hash -> nodeID
	virtualNodes  int               // number of virtual nodes per physical node
	totalKeys     int               // total number of keys assigned
	loadFactor    float64           // load factor for bounded loads
	keyAssignment map[string]string // key -> nodeID mapping (for tracking)
}

// Config holds configuration for the hash ring
type Config struct {
	VirtualNodes int     // Number of virtual nodes per physical node (default: 150)
	LoadFactor   float64 // Load factor for bounded loads (default: 1.25)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		VirtualNodes: 150,
		LoadFactor:   1.25,
	}
}

// NewHashRing creates a new consistent hash ring
func NewHashRing(config Config) *HashRing {
	if config.VirtualNodes <= 0 {
		config.VirtualNodes = 150
	}
	if config.LoadFactor <= 0 {
		config.LoadFactor = 1.25
	}

	return &HashRing{
		nodes:         make(map[string]*Node),
		ring:          make([]uint32, 0),
		ringMap:       make(map[uint32]string),
		virtualNodes:  config.VirtualNodes,
		loadFactor:    config.LoadFactor,
		keyAssignment: make(map[string]string),
	}
}

// hash generates a 32-bit hash using SHA-256
func (h *HashRing) hash(key string) uint32 {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	hashBytes := hasher.Sum(nil)
	return binary.BigEndian.Uint32(hashBytes[:4])
}

// AddNode adds a physical node to the hash ring
func (h *HashRing) AddNode(node *Node) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already exists", node.ID)
	}

	h.nodes[node.ID] = node

	// Add virtual nodes to the ring
	for i := 0; i < h.virtualNodes; i++ {
		virtualKey := fmt.Sprintf("%s#%d", node.ID, i)
		hashVal := h.hash(virtualKey)
		h.ring = append(h.ring, hashVal)
		h.ringMap[hashVal] = node.ID
	}

	// Sort the ring
	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})

	// Recalculate max loads for all nodes
	h.updateMaxLoads()

	// Rebalance existing keys if any
	h.rebalanceKeys()

	return nil
}

// RemoveNode removes a physical node from the hash ring
func (h *HashRing) RemoveNode(nodeID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	node, exists := h.nodes[nodeID]
	if !exists {
		return fmt.Errorf("node %s not found", nodeID)
	}

	// Remove virtual nodes from the ring
	newRing := make([]uint32, 0, len(h.ring))
	for _, hashVal := range h.ring {
		if h.ringMap[hashVal] != nodeID {
			newRing = append(newRing, hashVal)
		} else {
			delete(h.ringMap, hashVal)
		}
	}
	h.ring = newRing

	// Update total keys count
	h.totalKeys -= node.Load

	delete(h.nodes, nodeID)

	// Recalculate max loads
	h.updateMaxLoads()

	// Reassign keys that were on the removed node
	h.rebalanceKeys()

	return nil
}

// GetNode returns the node responsible for a given key
func (h *HashRing) GetNode(key string) (*Node, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	hashVal := h.hash(key)
	return h.getNodeForHash(hashVal), nil
}

// GetNodeWithBoundedLoad returns a node for the key respecting bounded loads
func (h *HashRing) GetNodeWithBoundedLoad(key string) (*Node, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.nodes) == 0 {
		return nil, fmt.Errorf("no nodes available")
	}

	hashVal := h.hash(key)

	// Try to find a node that can accept the key
	startIdx := h.search(hashVal)

	// Check if key already assigned
	if existingNodeID, exists := h.keyAssignment[key]; exists {
		if node, ok := h.nodes[existingNodeID]; ok {
			return node, nil
		}
		// Node no longer exists, will reassign
		delete(h.keyAssignment, key)
	}

	// Walk the ring to find an available node
	ringLen := len(h.ring)
	for i := 0; i < ringLen; i++ {
		idx := (startIdx + i) % ringLen
		nodeID := h.ringMap[h.ring[idx]]
		node := h.nodes[nodeID]

		if node.CanAcceptKey() {
			node.IncrementLoad()
			h.keyAssignment[key] = nodeID
			h.totalKeys++
			return node, nil
		}
	}

	// All nodes are at capacity, assign to the original node anyway
	// (This shouldn't happen with proper load factor, but we handle it)
	nodeID := h.ringMap[h.ring[startIdx]]
	node := h.nodes[nodeID]
	node.IncrementLoad()
	h.keyAssignment[key] = nodeID
	h.totalKeys++
	return node, nil
}

// RemoveKey removes a key assignment and updates load
func (h *HashRing) RemoveKey(key string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	nodeID, exists := h.keyAssignment[key]
	if !exists {
		return fmt.Errorf("key %s not found", key)
	}

	if node, ok := h.nodes[nodeID]; ok {
		node.DecrementLoad()
	}

	delete(h.keyAssignment, key)
	h.totalKeys--

	return nil
}

// getNodeForHash finds the node for a given hash value (without lock)
func (h *HashRing) getNodeForHash(hashVal uint32) *Node {
	idx := h.search(hashVal)
	nodeID := h.ringMap[h.ring[idx]]
	return h.nodes[nodeID]
}

// search performs binary search to find the first node >= hashVal
func (h *HashRing) search(hashVal uint32) int {
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hashVal
	})

	// Wrap around if we're past the end
	if idx >= len(h.ring) {
		idx = 0
	}

	return idx
}

// updateMaxLoads calculates and updates max load for all nodes
func (h *HashRing) updateMaxLoads() {
	if len(h.nodes) == 0 {
		return
	}

	// Calculate average load per node
	avgLoad := float64(h.totalKeys) / float64(len(h.nodes))

	// Max load = ceil(avgLoad * loadFactor)
	maxLoad := int(avgLoad*h.loadFactor + 0.5) // +0.5 for ceiling
	if maxLoad == 0 {
		maxLoad = 1 // Minimum of 1
	}

	// Update all nodes
	for _, node := range h.nodes {
		node.MaxLoad = maxLoad
	}
}

// rebalanceKeys reassigns keys that need rebalancing
func (h *HashRing) rebalanceKeys() {
	// Reset all loads
	for _, node := range h.nodes {
		node.ResetLoad()
	}

	// Reassign all keys
	newAssignment := make(map[string]string)
	for key := range h.keyAssignment {
		hashVal := h.hash(key)
		nodeID := h.getNodeForHash(hashVal).ID
		h.nodes[nodeID].IncrementLoad()
		newAssignment[key] = nodeID
	}

	h.keyAssignment = newAssignment
}

// GetNodes returns all nodes in the ring
func (h *HashRing) GetNodes() []*Node {
	h.mu.RLock()
	defer h.mu.RUnlock()

	nodes := make([]*Node, 0, len(h.nodes))
	for _, node := range h.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Stats returns statistics about the hash ring
func (h *HashRing) Stats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	nodeStats := make([]map[string]interface{}, 0, len(h.nodes))
	for _, node := range h.nodes {
		nodeStats = append(nodeStats, map[string]interface{}{
			"id":       node.ID,
			"host":     node.Host,
			"load":     node.Load,
			"max_load": node.MaxLoad,
		})
	}

	return map[string]interface{}{
		"total_nodes":   len(h.nodes),
		"total_keys":    h.totalKeys,
		"virtual_nodes": h.virtualNodes,
		"load_factor":   h.loadFactor,
		"ring_size":     len(h.ring),
		"nodes":         nodeStats,
	}
}
