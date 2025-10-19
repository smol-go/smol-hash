package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/smol-go/smol-hash/pkg/consistenthash"
)

func main() {
	fmt.Println("🔷 smol-hash - Consistent Hashing with Bounded Loads")
	fmt.Println("====================================================")

	// Create a hash ring with default config
	config := consistenthash.DefaultConfig()
	ring := consistenthash.NewHashRing(config)

	fmt.Printf("Configuration:\n")
	fmt.Printf("  Virtual Nodes: %d\n", config.VirtualNodes)
	fmt.Printf("  Load Factor: %.2f\n\n", config.LoadFactor)

	// Add some nodes
	fmt.Println("Adding nodes to the ring...")
	nodes := []struct {
		id   string
		host string
	}{
		{"server1", "192.168.1.1:8080"},
		{"server2", "192.168.1.2:8080"},
		{"server3", "192.168.1.3:8080"},
		{"server4", "192.168.1.4:8080"},
		{"server5", "192.168.1.5:8080"},
	}

	for _, n := range nodes {
		node := consistenthash.NewNode(n.id, n.host)
		err := ring.AddNode(node)
		if err != nil {
			fmt.Printf("❌ Error adding node %s: %v\n", n.id, err)
			os.Exit(1)
		}
		fmt.Printf("  ✓ Added %s (%s)\n", n.id, n.host)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Scenario 1: Standard Key Distribution")
	fmt.Println(strings.Repeat("=", 50))

	// Distribute some keys
	fmt.Println("\nDistributing 50 keys using standard GetNode()...")
	distribution := make(map[string]int)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("user:%d", i)
		node, err := ring.GetNode(key)
		if err != nil {
			fmt.Printf("❌ Error getting node for %s: %v\n", key, err)
			continue
		}
		distribution[node.ID]++
		if i < 5 {
			fmt.Printf("  %s -> %s (%s)\n", key, node.ID, node.Host)
		}
	}

	if len(distribution) > 5 {
		fmt.Printf("  ... (45 more keys)\n")
	}

	fmt.Println("\nDistribution without bounded loads:")
	for nodeID, count := range distribution {
		fmt.Printf("  %s: %d keys (%.1f%%)\n", nodeID, count, float64(count)/50*100)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Scenario 2: Bounded Load Distribution")
	fmt.Println(strings.Repeat("=", 50))

	// Create a new ring for bounded load test
	ring2 := consistenthash.NewHashRing(config)
	for _, n := range nodes {
		node := consistenthash.NewNode(n.id, n.host)
		ring2.AddNode(node)
	}

	fmt.Println("\nDistributing 50 keys with bounded loads...")
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("user:%d", i)
		node, err := ring2.GetNodeWithBoundedLoad(key)
		if err != nil {
			fmt.Printf("❌ Error getting node for %s: %v\n", key, err)
			continue
		}
		if i < 5 {
			fmt.Printf("  %s -> %s (%s) [Load: %d/%d]\n",
				key, node.ID, node.Host, node.Load, node.MaxLoad)
		}
	}

	if len(distribution) > 5 {
		fmt.Printf("  ... (45 more keys)\n")
	}

	fmt.Println("\nDistribution with bounded loads:")
	allNodes := ring2.GetNodes()
	for _, node := range allNodes {
		percentage := float64(node.Load) / 50 * 100
		bar := generateBar(node.Load, node.MaxLoad)
		fmt.Printf("  %s: %d/%d keys (%.1f%%) %s\n",
			node.ID, node.Load, node.MaxLoad, percentage, bar)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Scenario 3: Node Failure Simulation")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\nRemoving server3 to simulate failure...")
	err := ring2.RemoveNode("server3")
	if err != nil {
		fmt.Printf("❌ Error removing node: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✓ server3 removed")

	fmt.Println("\nRedistribution after node removal:")
	allNodes = ring2.GetNodes()
	totalKeys := 0
	for _, node := range allNodes {
		totalKeys += node.Load
		percentage := float64(node.Load) / float64(totalKeys) * 100
		bar := generateBar(node.Load, node.MaxLoad)
		fmt.Printf("  %s: %d/%d keys (%.1f%%) %s\n",
			node.ID, node.Load, node.MaxLoad, percentage, bar)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Statistics")
	fmt.Println(strings.Repeat("=", 50))

	stats := ring2.Stats()
	jsonStats, _ := json.MarshalIndent(stats, "", "  ")
	fmt.Println(string(jsonStats))

	fmt.Println("\n✅ Demo completed successfully!")
}

// generateBar creates a simple text-based progress bar
func generateBar(current, max int) string {
	if max == 0 {
		return ""
	}
	barLength := 20
	filled := int(float64(current) / float64(max) * float64(barLength))
	if filled > barLength {
		filled = barLength
	}

	bar := "["
	for i := 0; i < barLength; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"
	return bar
}
