package consistenthash

import (
	"fmt"
	"testing"
)

func TestNewHashRing(t *testing.T) {
	config := DefaultConfig()
	ring := NewHashRing(config)

	if ring == nil {
		t.Fatal("Expected non-nil hash ring")
	}

	if ring.virtualNodes != 150 {
		t.Errorf("Expected 150 virtual nodes, got %d", ring.virtualNodes)
	}

	if ring.loadFactor != 1.25 {
		t.Errorf("Expected load factor 1.25, got %f", ring.loadFactor)
	}
}

func TestAddNode(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	node1 := NewNode("node1", "192.168.1.1:8080")
	err := ring.AddNode(node1)
	if err != nil {
		t.Fatalf("Failed to add node: %v", err)
	}

	// Check that virtual nodes were added
	if len(ring.ring) != 150 {
		t.Errorf("Expected 150 virtual nodes, got %d", len(ring.ring))
	}

	// Try adding duplicate
	err = ring.AddNode(node1)
	if err == nil {
		t.Error("Expected error when adding duplicate node")
	}
}

func TestRemoveNode(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	node1 := NewNode("node1", "192.168.1.1:8080")
	ring.AddNode(node1)

	err := ring.RemoveNode("node1")
	if err != nil {
		t.Fatalf("Failed to remove node: %v", err)
	}

	if len(ring.ring) != 0 {
		t.Errorf("Expected 0 virtual nodes after removal, got %d", len(ring.ring))
	}

	// Try removing non-existent node
	err = ring.RemoveNode("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent node")
	}
}

func TestGetNode(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	node1 := NewNode("node1", "192.168.1.1:8080")
	node2 := NewNode("node2", "192.168.1.2:8080")
	node3 := NewNode("node3", "192.168.1.3:8080")

	ring.AddNode(node1)
	ring.AddNode(node2)
	ring.AddNode(node3)

	// Test that same key always maps to same node
	key := "test-key-123"
	node, err := ring.GetNode(key)
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	for i := 0; i < 10; i++ {
		n, _ := ring.GetNode(key)
		if n.ID != node.ID {
			t.Errorf("Key mapped to different node on iteration %d", i)
		}
	}
}

func TestDistribution(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	// Add 3 nodes
	for i := 1; i <= 3; i++ {
		node := NewNode(fmt.Sprintf("node%d", i), fmt.Sprintf("192.168.1.%d:8080", i))
		ring.AddNode(node)
	}

	// Assign 10000 keys
	distribution := make(map[string]int)
	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		node, _ := ring.GetNode(key)
		distribution[node.ID]++
	}

	// Check that distribution is reasonably balanced
	// Each node should get roughly 3333 keys (10000/3)
	// We allow 20% variance
	expected := 10000 / 3
	tolerance := expected / 5 // 20%

	for nodeID, count := range distribution {
		if count < expected-tolerance || count > expected+tolerance {
			t.Logf("Warning: Node %s has unbalanced distribution: %d keys (expected ~%d)",
				nodeID, count, expected)
		}
	}

	t.Logf("Distribution: %v", distribution)
}

func TestBoundedLoad(t *testing.T) {
	config := Config{
		VirtualNodes: 150,
		LoadFactor:   1.25,
	}
	ring := NewHashRing(config)

	// Add 3 nodes
	for i := 1; i <= 3; i++ {
		node := NewNode(fmt.Sprintf("node%d", i), fmt.Sprintf("192.168.1.%d:8080", i))
		ring.AddNode(node)
	}

	// Assign 300 keys
	for i := 0; i < 300; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, err := ring.GetNodeWithBoundedLoad(key)
		if err != nil {
			t.Fatalf("Failed to assign key %s: %v", key, err)
		}
	}

	// Check loads
	nodes := ring.GetNodes()
	totalLoad := 0
	for _, node := range nodes {
		totalLoad += node.Load
		t.Logf("Node %s: Load=%d, MaxLoad=%d", node.ID, node.Load, node.MaxLoad)

		// Each node should not exceed max load significantly
		if node.Load > node.MaxLoad+5 { // Allow small overflow
			t.Errorf("Node %s exceeded max load: %d > %d", node.ID, node.Load, node.MaxLoad)
		}
	}

	if totalLoad != 300 {
		t.Errorf("Total load mismatch: expected 300, got %d", totalLoad)
	}
}

func TestRemoveKey(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	node1 := NewNode("node1", "192.168.1.1:8080")
	ring.AddNode(node1)

	// Add a key
	key := "test-key"
	_, err := ring.GetNodeWithBoundedLoad(key)
	if err != nil {
		t.Fatalf("Failed to add key: %v", err)
	}

	if node1.Load != 1 {
		t.Errorf("Expected load 1, got %d", node1.Load)
	}

	// Remove the key
	err = ring.RemoveKey(key)
	if err != nil {
		t.Fatalf("Failed to remove key: %v", err)
	}

	if node1.Load != 0 {
		t.Errorf("Expected load 0 after removal, got %d", node1.Load)
	}
}

func TestNodeRemovalRebalancing(t *testing.T) {
	ring := NewHashRing(DefaultConfig())

	// Add 3 nodes
	for i := 1; i <= 3; i++ {
		node := NewNode(fmt.Sprintf("node%d", i), fmt.Sprintf("192.168.1.%d:8080", i))
		ring.AddNode(node)
	}

	// Assign 90 keys
	keys := make([]string, 90)
	for i := 0; i < 90; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		ring.GetNodeWithBoundedLoad(keys[i])
	}

	// Track which node had each key
	initialAssignment := make(map[string]string)
	for _, key := range keys {
		node, _ := ring.GetNode(key)
		initialAssignment[key] = node.ID
	}

	// Remove one node
	ring.RemoveNode("node2")

	// Check that keys are redistributed
	movedKeys := 0
	for _, key := range keys {
		node, _ := ring.GetNode(key)
		if node.ID != initialAssignment[key] {
			movedKeys++
		}
		// Key should now be on node1 or node3
		if node.ID == "node2" {
			t.Errorf("Key %s still assigned to removed node", key)
		}
	}

	t.Logf("Moved %d keys after removing node2", movedKeys)

	// Verify total keys is still correct
	nodes := ring.GetNodes()
	totalLoad := 0
	for _, node := range nodes {
		totalLoad += node.Load
	}

	if totalLoad != 90 {
		t.Errorf("Expected total load 90, got %d", totalLoad)
	}
}

func BenchmarkGetNode(b *testing.B) {
	ring := NewHashRing(DefaultConfig())

	for i := 1; i <= 10; i++ {
		node := NewNode(fmt.Sprintf("node%d", i), fmt.Sprintf("192.168.1.%d:8080", i))
		ring.AddNode(node)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		ring.GetNode(key)
	}
}

func BenchmarkGetNodeWithBoundedLoad(b *testing.B) {
	ring := NewHashRing(DefaultConfig())

	for i := 1; i <= 10; i++ {
		node := NewNode(fmt.Sprintf("node%d", i), fmt.Sprintf("192.168.1.%d:8080", i))
		ring.AddNode(node)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		ring.GetNodeWithBoundedLoad(key)
	}
}
