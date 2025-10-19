package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/smol-go/smol-hash/pkg/consistenthash"
)

// Request represents an incoming request
type Request struct {
	ID        string
	SessionID string
	Timestamp time.Time
}

// Backend represents a backend server
type Backend struct {
	*consistenthash.Node
	RequestCount int
	LastRequest  time.Time
}

// LoadBalancer manages request distribution
type LoadBalancer struct {
	ring     *consistenthash.HashRing
	backends map[string]*Backend
}

func NewLoadBalancer() *LoadBalancer {
	config := consistenthash.Config{
		VirtualNodes: 150,
		LoadFactor:   1.25,
	}

	return &LoadBalancer{
		ring:     consistenthash.NewHashRing(config),
		backends: make(map[string]*Backend),
	}
}

func (lb *LoadBalancer) AddBackend(id, host string) error {
	node := consistenthash.NewNode(id, host)
	backend := &Backend{
		Node:         node,
		RequestCount: 0,
		LastRequest:  time.Now(),
	}

	if err := lb.ring.AddNode(node); err != nil {
		return err
	}

	lb.backends[id] = backend
	return nil
}

func (lb *LoadBalancer) RemoveBackend(id string) error {
	if err := lb.ring.RemoveNode(id); err != nil {
		return err
	}
	delete(lb.backends, id)
	return nil
}

func (lb *LoadBalancer) RouteRequest(req Request) (*Backend, error) {
	// Use session ID for consistent routing
	node, err := lb.ring.GetNodeWithBoundedLoad(req.SessionID)
	if err != nil {
		return nil, err
	}

	backend := lb.backends[node.ID]
	backend.RequestCount++
	backend.LastRequest = time.Now()

	return backend, nil
}

func (lb *LoadBalancer) PrintStats() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Load Balancer Statistics")
	fmt.Println(strings.Repeat("=", 60))

	stats := lb.ring.Stats()
	fmt.Printf("Total Backends: %d\n", stats["total_nodes"])
	fmt.Printf("Total Sessions: %d\n", stats["total_keys"])
	fmt.Printf("Virtual Nodes per Backend: %d\n", stats["virtual_nodes"])
	fmt.Printf("Load Factor: %.2f\n\n", stats["load_factor"])

	fmt.Println("Backend Status:")
	fmt.Println("  " + strings.Repeat("-", 56))
	fmt.Printf("  %-12s %-20s %-10s %-10s\n", "Backend", "Host", "Sessions", "Requests")
	fmt.Println("  " + strings.Repeat("-", 56))

	for _, backend := range lb.backends {
		fmt.Printf("  %-12s %-20s %-10d %-10d\n",
			backend.ID,
			backend.Host,
			backend.Load,
			backend.RequestCount,
		)
	}
	fmt.Println("  " + strings.Repeat("-", 56))
}

func main() {
	fmt.Println("smol-hash Load Balancer Demo")
	fmt.Println("=" + strings.Repeat("=", 59))

	// Create load balancer
	lb := NewLoadBalancer()

	// Add backend servers
	fmt.Println("\nInitializing backend servers...")
	backends := []struct {
		id   string
		host string
	}{
		{"backend-1", "10.0.1.1:8080"},
		{"backend-2", "10.0.1.2:8080"},
		{"backend-3", "10.0.1.3:8080"},
		{"backend-4", "10.0.1.4:8080"},
	}

	for _, b := range backends {
		if err := lb.AddBackend(b.id, b.host); err != nil {
			log.Fatalf("Failed to add backend %s: %v", b.id, err)
		}
		fmt.Printf("%s online at %s\n", b.id, b.host)
	}

	// Simulate incoming requests
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Simulating Traffic")
	fmt.Println(strings.Repeat("=", 60))

	rand.Seed(time.Now().UnixNano())

	// Generate sessions
	numSessions := 100
	numRequestsPerSession := rand.Intn(5) + 1

	fmt.Printf("\nGenerating traffic: %d sessions, ~%d requests each\n",
		numSessions, numRequestsPerSession)

	totalRequests := 0
	sessionDistribution := make(map[string]int)

	for session := 0; session < numSessions; session++ {
		sessionID := fmt.Sprintf("session-%d", session)

		// Each session sends multiple requests
		requests := rand.Intn(5) + 1
		for r := 0; r < requests; r++ {
			req := Request{
				ID:        fmt.Sprintf("req-%d-%d", session, r),
				SessionID: sessionID,
				Timestamp: time.Now(),
			}

			backend, err := lb.RouteRequest(req)
			if err != nil {
				log.Printf("Failed to route request: %v", err)
				continue
			}

			sessionDistribution[backend.ID]++
			totalRequests++

			// Show first few routings
			if totalRequests <= 5 {
				fmt.Printf("  %s (%s) -> %s\n",
					req.ID, req.SessionID, backend.ID)
			}
		}
	}

	if totalRequests > 5 {
		fmt.Printf("...(%d more requests)\n", totalRequests-5)
	}

	// Print statistics
	lb.PrintStats()

	// Simulate backend failure
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Simulating Backend Failure")
	fmt.Println(strings.Repeat("=", 60))

	failedBackend := "backend-2"
	fmt.Printf("\n%s has failed! Removing from pool...\n", failedBackend)

	if err := lb.RemoveBackend(failedBackend); err != nil {
		log.Fatalf("Failed to remove backend: %v", err)
	}
	fmt.Printf("%s removed successfully\n", failedBackend)

	// Simulate more traffic after failure
	fmt.Println("\nRouting new traffic to remaining backends...")
	for i := 0; i < 20; i++ {
		sessionID := fmt.Sprintf("new-session-%d", i)
		req := Request{
			ID:        fmt.Sprintf("new-req-%d", i),
			SessionID: sessionID,
			Timestamp: time.Now(),
		}

		backend, err := lb.RouteRequest(req)
		if err != nil {
			log.Printf("Failed to route request: %v", err)
			continue
		}

		if i < 3 {
			fmt.Printf("  %s -> %s\n", req.SessionID, backend.ID)
		}
	}

	if totalRequests > 3 {
		fmt.Printf("...(%d more requests)\n", totalRequests-3)
	}

	// Final statistics
	lb.PrintStats()

	// Demonstrate session affinity
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Session Affinity Test")
	fmt.Println(strings.Repeat("=", 60))

	testSession := "sticky-session-123"
	fmt.Printf("\nSending 10 requests from %s:\n", testSession)

	var firstBackend string
	consistent := true

	for i := 0; i < 10; i++ {
		req := Request{
			ID:        fmt.Sprintf("sticky-%d", i),
			SessionID: testSession,
			Timestamp: time.Now(),
		}

		backend, _ := lb.RouteRequest(req)

		if i == 0 {
			firstBackend = backend.ID
		} else if backend.ID != firstBackend {
			consistent = false
		}

		if i < 3 {
			fmt.Printf("Request %d -> %s\n", i+1, backend.ID)
		}
	}

	if consistent {
		fmt.Printf("...\nAll requests routed to %s (session affinity maintained)\n",
			firstBackend)
	} else {
		fmt.Println("Session affinity broken!")
	}

	fmt.Println("\nLoad balancer demo completed!")
}
