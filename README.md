# smol-hash

A beginner-friendly, lightweight implementation of consistent hashing with bounded loads in Go.

## What is Consistent Hashing?

Consistent hashing is a distributed hashing technique that minimizes key redistribution when nodes are added or removed from a cluster. This implementation adds **bounded loads** to prevent any single node from becoming overloaded.

## Features

- 🎯 **Simple API** - Easy to understand and use
- ⚖️ **Load Balancing** - Bounded loads prevent hotspots
- 🔄 **Dynamic Scaling** - Add/remove nodes without major redistribution
- 🔒 **Thread-Safe** - Safe for concurrent use
- 📊 **Statistics** - Track load distribution across nodes
- 🪶 **Lightweight** - No external dependencies

## Installation

```bash
git clone https://github.com/smol-go/smol-hash.git
cd smol-hash
```

## Quick Start

```go
package main

import (
    "fmt"
)

func main() {
    // Create hash ring with 150 virtual nodes per server
    // and max load of 1.25x average (125%)
    ch := NewConsistentHash(150, 1.25)

    // Add servers
    ch.AddNode("server1")
    ch.AddNode("server2")
    ch.AddNode("server3")

    // Get node for a key
    node, err := ch.GetNode("user:12345")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Key assigned to: %s\n", node)

    // View load distribution
    stats := ch.GetStats()
    for server, load := range stats {
        fmt.Printf("%s: %d keys\n", server, load)
    }
}
```

## How It Works

### 1. Virtual Nodes (Replicas)
Each physical server is mapped to multiple points on the hash ring (virtual nodes). More replicas = better distribution.

```go
// 150 replicas provides good balance between
// distribution quality and memory usage
ch := NewConsistentHash(150, 1.25)
```

### 2. Bounded Loads
Prevents overloading by limiting each node to a maximum load:

```
max_load = average_load × load_factor
```

For example, with 3 nodes, 15 keys, and load factor 1.25:
- Average load: 15 / 3 = 5 keys
- Max load per node: 5 × 1.25 = 6.25 → 6 keys

### 3. Key Assignment
When you call `GetNode(key)`:
1. Hash the key to find position on ring
2. Find closest node clockwise
3. If that node is at max capacity, try next node
4. Continue until finding available node

## API Reference

### Creating a Hash Ring

```go
NewConsistentHash(replicas int, loadFactor float64) *ConsistentHash
```

- `replicas`: Number of virtual nodes per physical node (50-200 recommended)
- `loadFactor`: Max load multiplier (1.25 = 125% of average)

### Managing Nodes

```go
// Add a node to the ring
AddNode(nodeName string)

// Remove a node from the ring
RemoveNode(nodeName string)
```

### Key Operations

```go
// Get the node responsible for a key
GetNode(key string) (string, error)

// Release a key (decrement load counter)
ReleaseKey(key string) error
```

### Statistics

```go
// Get current load for all nodes
GetStats() map[string]int
```

## Configuration Guide

### Choosing Replicas

| Replicas | Distribution | Memory | Use Case |
|----------|-------------|---------|----------|
| 10-50 | Fair | Low | Small clusters, testing |
| 100-200 | Good | Medium | Production (recommended) |
| 500+ | Excellent | High | Large clusters, strict balance |

### Choosing Load Factor

| Load Factor | Behavior | Use Case |
|------------|----------|----------|
| 1.0 | Strict equality | Academic/testing |
| 1.25 | Balanced (default) | Most production use cases |
| 1.5+ | More relaxed | High throughput scenarios |

## Use Cases

- **Cache Distribution**: Distribute cache keys across multiple Redis/Memcached instances
- **Database Sharding**: Route queries to appropriate database shards
- **Load Balancing**: Distribute requests across API servers
- **Service Discovery**: Map services to instances in microservices
- **CDN Routing**: Route content requests to edge servers

## Example: Cache Distribution

```go
package main

import (
    "fmt"
)

type CacheRouter struct {
    hash *ConsistentHash
}

func NewCacheRouter() *CacheRouter {
    ch := NewConsistentHash(150, 1.25)
    
    // Add cache servers
    ch.AddNode("cache-1.example.com:6379")
    ch.AddNode("cache-2.example.com:6379")
    ch.AddNode("cache-3.example.com:6379")
    
    return &CacheRouter{hash: ch}
}

func (cr *CacheRouter) GetServer(key string) (string, error) {
    return cr.hash.GetNode(key)
}

func main() {
    router := NewCacheRouter()
    
    // Route user sessions to cache servers
    userID := "user:12345"
    server, _ := router.GetServer(userID)
    fmt.Printf("User %s -> %s\n", userID, server)
}
```

## Performance Characteristics

- **Time Complexity**:
  - Add/Remove Node: O(R log N) where R = replicas, N = total nodes
  - Get Node: O(log N + N) worst case, typically O(log N)
  
- **Space Complexity**: O(R × N) for virtual nodes