package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	rbolt "github.com/hashicorp/raft-boltdb"
)

// Config struct handles configuration for a node
type Config struct {
	BindAddress    string
	NodeIdentifier string
	JoinAddress    string
	DataDir        string
	Bootstrap      bool
}

// NewRStorage initiates a new RStorage node
func NewRStorage(config *Config) (*RStorage, error) {
	rstorage := RStorage{
		storage: map[string]string{},
		config:  *config,
	}

	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, err
	}

	logger := log.New(os.Stdout, "[raft]", 0)

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(config.NodeIdentifier)
	raftConfig.Logger = logger
	transport, err := raftTransport(config.BindAddress)
	if err != nil {
		return nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(config.DataDir, 1, os.Stdout)
	if err != nil {
		return nil, err
	}

	logStore, err := rbolt.NewBoltStore(filepath.Join(config.DataDir, "raft-log.bolt"))
	if err != nil {
		return nil, err
	}

	stableStore, err := rbolt.NewBoltStore(filepath.Join(config.DataDir, "raft-stable.bolt"))
	if err != nil {
		return nil, err
	}

	raftNode, err := raft.NewRaft(
		raftConfig,
		&rstorage,
		logStore,
		stableStore,
		snapshotStore,
		transport,
	)
	if err != nil {
		return nil, err
	}

	if config.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		raftNode.BootstrapCluster(configuration)
	}

	rstorage.RaftNode = raftNode

	return &rstorage, nil
}

func raftTransport(bindAddr string) (*raft.NetworkTransport, error) {
	address, err := net.ResolveTCPAddr("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	logger := log.New(os.Stdout, "[raft]", 0)
	transport, err := raft.NewTCPTransportWithLogger(bindAddr, address, 3, 10*time.Second, logger)
	if err != nil {
		return nil, err
	}

	return transport, nil
}

// GetClusterServers returns all cluster's servers
func (s *RStorage) GetClusterServers() ([]raft.Server, error) {
	confugurationFuture := s.RaftNode.GetConfiguration()
	if err := confugurationFuture.Error(); err != nil {
		log.Printf("[ERROR] Reading Raft configuration error: %+v", err)
		return nil, err
	}

	return confugurationFuture.Configuration().Servers, nil
}

// AddVoter joins a new voter to a cluster
// must be called only on a leader
func (s *RStorage) AddVoter(address string) error {
	log.Printf("[INFO] trying to add new voter at [%s] to the cluster", address)
	addFuture := s.RaftNode.AddVoter(raft.ServerID(address), raft.ServerAddress(address), 0, 0)
	if err := addFuture.Error(); err != nil {
		log.Printf("[ERROR] cant join to the cluster: %v", err)
		return err
	}
	return nil
}

// JoinCluster sends a POST request to "join" address
// to ask the cluster leader join this node as a voter
func (s *RStorage) JoinCluster(leaderHTTPAddress string) error {
	servers, err := s.GetClusterServers()
	alreadyInCluster := (err == nil && len(servers) > 1)
	if alreadyInCluster {
		log.Printf("[INFO] Node already in the cluster, skipping /cluster/join/ POST request to the leader")
		return nil
	}

	body, err := json.Marshal(map[string]string{"address": s.config.BindAddress})
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/cluster/join/", leaderHTTPAddress),
		"application-type/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Leader status code is not 200: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	return nil
}
