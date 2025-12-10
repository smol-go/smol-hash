package main

import (
	"fmt"
	"hash/fnv"
	"slices"
	"sort"
	"sync"
)

// ConsistentHash implements consistent hashing with bounded loads
type ConsistentHash struct {
	mu         sync.RWMutex
	ring       []uint32
	ringMap    map[uint32]string
	nodes      map[string]*NodeInfo
	replicas   int
	loadFactor float64
}

// NodeInfo stores information about each node
type NodeInfo struct {
	name string
	load int // Current number of keys assigned to this node
}

// NewConsistentHash creates a new consistent hash ring
func NewConsistentHash(replicas int, loadFactor float64) *ConsistentHash {
	return &ConsistentHash{
		ringMap:    make(map[uint32]string),
		nodes:      make(map[string]*NodeInfo),
		replicas:   replicas,
		loadFactor: loadFactor,
	}
}

// hash generates a hash value for a given key
func (ch *ConsistentHash) hash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// AddNode adds a new node to the hash ring
func (ch *ConsistentHash) AddNode(nodeName string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Check if node already exists
	if _, exists := ch.nodes[nodeName]; exists {
		return
	}

	// Add node info
	ch.nodes[nodeName] = &NodeInfo{
		name: nodeName,
		load: 0,
	}

	// Add virtual nodes (replicas) to the ring
	for i := range ch.replicas {
		virtualKey := fmt.Sprintf("%s#%d", nodeName, i)
		hashVal := ch.hash(virtualKey)
		ch.ring = append(ch.ring, hashVal)
		ch.ringMap[hashVal] = nodeName
	}

	// Sort the ring
	slices.Sort(ch.ring)
}

// RemoveNode removes a node from the hash ring
func (ch *ConsistentHash) RemoveNode(nodeName string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if _, exists := ch.nodes[nodeName]; !exists {
		return
	}

	// Remove virtual nodes from the ring
	for i := 0; i < ch.replicas; i++ {
		virtualKey := fmt.Sprintf("%s#%d", nodeName, i)
		hashVal := ch.hash(virtualKey)
		delete(ch.ringMap, hashVal)

		// Remove from ring slice
		idx := ch.search(hashVal)
		ch.ring = append(ch.ring[:idx], ch.ring[idx+1:]...)
	}

	delete(ch.nodes, nodeName)
}

// search finds the index of a hash value in the ring using binary search
func (ch *ConsistentHash) search(hashVal uint32) int {
	return sort.Search(len(ch.ring), func(i int) bool {
		return ch.ring[i] >= hashVal
	})
}

// GetNode returns the node responsible for a given key with bounded load
func (ch *ConsistentHash) GetNode(key string) (string, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.ring) == 0 {
		return "", fmt.Errorf("no nodes available")
	}

	// Calculate average load and max allowed load
	totalNodes := len(ch.nodes)
	totalLoad := 0
	for _, node := range ch.nodes {
		totalLoad += node.load
	}

	avgLoad := float64(totalLoad+1) / float64(totalNodes) // +1 for the new key
	maxLoad := int(avgLoad * ch.loadFactor)

	// Hash the key
	hashVal := ch.hash(key)
	idx := ch.search(hashVal)

	// Search for a node with available capacity
	// Start from the closest node and wrap around if necessary
	for i := range ch.ring {
		currIdx := (idx + i) % len(ch.ring)
		nodeName := ch.ringMap[ch.ring[currIdx]]
		node := ch.nodes[nodeName]

		// Check if this node is under the load limit
		if node.load < maxLoad || maxLoad == 0 {
			node.load++
			return nodeName, nil
		}
	}

	// If all nodes are at capacity, return the originally hashed node
	nodeName := ch.ringMap[ch.ring[idx%len(ch.ring)]]
	ch.nodes[nodeName].load++
	return nodeName, nil
}

// ReleaseKey decrements the load for a key's assigned node
func (ch *ConsistentHash) ReleaseKey(key string) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.ring) == 0 {
		return fmt.Errorf("no nodes available")
	}

	hashVal := ch.hash(key)
	idx := ch.search(hashVal)
	if idx >= len(ch.ring) {
		idx = 0
	}

	nodeName := ch.ringMap[ch.ring[idx]]
	node := ch.nodes[nodeName]

	if node.load > 0 {
		node.load--
	}

	return nil
}

// GetStats returns load statistics for all nodes
func (ch *ConsistentHash) GetStats() map[string]int {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	stats := make(map[string]int)
	for name, node := range ch.nodes {
		stats[name] = node.load
	}
	return stats
}

// Example usage
func main() {
	// Create a consistent hash with 150 virtual nodes per physical node
	// and a load factor of 1.25 (nodes can have up to 125% of average load)
	ch := NewConsistentHash(150, 1.25)

	// Add some nodes
	ch.AddNode("server1")
	ch.AddNode("server2")
	ch.AddNode("server3")

	// Simulate adding keys
	keys := []string{
		"user:1001", "user:1002", "user:1003", "user:1004", "user:1005",
		"user:1006", "user:1007", "user:1008", "user:1009", "user:1010",
		"session:abc", "session:def", "session:ghi", "cache:key1", "cache:key2",
	}

	fmt.Println("Assigning keys to nodes:")
	for _, key := range keys {
		node, err := ch.GetNode(key)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Key '%s' -> %s\n", key, node)
	}

	// Print load distribution
	fmt.Println("\nLoad distribution:")
	stats := ch.GetStats()
	for node, load := range stats {
		fmt.Printf("%s: %d keys\n", node, load)
	}

	// Add a new node and see redistribution
	fmt.Println("\nAdding server4...")
	ch.AddNode("server4")

	// Assign more keys
	moreKeys := []string{"user:2001", "user:2002", "user:2003", "user:2004", "user:2005"}
	for _, key := range moreKeys {
		node, _ := ch.GetNode(key)
		fmt.Printf("Key '%s' -> %s\n", key, node)
	}

	// Print updated load distribution
	fmt.Println("\nUpdated load distribution:")
	stats = ch.GetStats()
	for node, load := range stats {
		fmt.Printf("%s: %d keys\n", node, load)
	}
}
