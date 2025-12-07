package smolhash

import "errors"

var (
	ErrNoServers            = errors.New("no servers available")
	ErrServerNotFound       = errors.New("server not found")
	ErrAllServersOverloaded = errors.New("all servers are overloaded")
)

type Server struct {
	ID       string
	Load     int64
	Metadata map[string]interface{}
}

type virtualNode struct {
	hash     uint64
	serverId string
}

type Stats struct {
	ServerCount          int
	VirtualNodeCount     int
	TotalLoad            int64
	AverageLoadPerServer float64
	MaxLoadPerServer     int64
	LoadBalanceFactor    float64
	ServerLoads          map[string]int64
}

type HashFunc interface {
	Sum64(data []byte) uint64
}

type Ring interface {
	AddServer(serverId string) error
	RemoveServer(serverId string) error
	GetServer(key string) (string, error)
	GetServerWithLoad(key string) (string, error)
	GetServerLoad(serverID string) (int64, error)
	GetServers() []string
	IncrementLoad(serverID string, amount int64) error
	DecrementLoad(serverID string, amount int64) error
	GetMaxLoad() int64
	SetLoadBalanceFactor(epsilon float64)
}
