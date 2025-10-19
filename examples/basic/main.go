package main

import (
	"fmt"
	"log"

	"github.com/smol-go/smol-hash/pkg/consistenthash"
)

func main() {
	// Create a hash ring with custom configuration
	config := consistenthash.Config{
		VirtualNodes: 100, // Lower number for this example
		LoadFactor:   1.5, // Allow 50% more load than average
	}

	ring := consistenthash.NewHashRing(config)

	// Add nodes
	fmt.Println("Adding nodes...")
	for i := 1; i <= 3; i++ {
		node := consistenthash.NewNode(
			fmt.Sprintf("cache-%d", i),
			fmt.Sprintf("10.0.0.%d:6379", i),
		)

		if err := ring.AddNode(node); err != nil {
			log.Fatalf("Failed to add node: %v", err)
		}
		fmt.Printf("  Added: %s\n", node.ID)
	}

	// Simple key lookup
	fmt.Println("\nLookup examples:")
	keys := []string{"user:1001", "session:abc123", "product:42", "cart:xyz"}

	for _, key := range keys {
		node, err := ring.GetNode(key)
		if err != nil {
			log.Printf("Error looking up %s: %v", key, err)
			continue
		}
		fmt.Printf("  %s -> %s (%s)\n", key, node.ID, node.Host)
	}

	// Demonstrate consistency - same key always goes to same node
	fmt.Println("\nConsistency test:")
	testKey := "user:9999"
	firstNode, _ := ring.GetNode(testKey)
	fmt.Printf("First lookup: %s -> %s\n", testKey, firstNode.ID)

	for i := range 5 {
		node, _ := ring.GetNode(testKey)
		if node.ID != firstNode.ID {
			fmt.Printf("Inconsistency detected at iteration %d!\n", i)
		}
	}
	fmt.Printf("Key consistently maps to %s\n", firstNode.ID)

	// Demonstrate bounded load
	fmt.Println("\nBounded load test:")
	fmt.Println("Assigning 15 keys with bounded load...")

	for i := range 15 {
		key := fmt.Sprintf("data:%d", i)
		node, err := ring.GetNodeWithBoundedLoad(key)
		if err != nil {
			log.Printf("Error assigning %s: %v", key, err)
			continue
		}
		if i < 3 {
			fmt.Printf("%s -> %s (load: %d/%d)\n",
				key, node.ID, node.Load, node.MaxLoad)
		}
	}

	// Show final distribution
	fmt.Println("\nFinal load distribution:")
	for _, node := range ring.GetNodes() {
		fmt.Printf("    %s: %d/%d keys\n", node.ID, node.Load, node.MaxLoad)
	}
}
