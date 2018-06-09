package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jessevdk/go-flags"

	"github.com/alexander-akhmetov/raft-example/src/node"
	"github.com/alexander-akhmetov/raft-example/src/server"
)

// Opts represents command line options
type Opts struct {
	BindAddress string `long:"bind" env:"BIND" default:"127.0.0.1:3000" description:"ip:port to bind for a node"`
	JoinAddress string `long:"join" env:"JOIN" default:"" description:"ip:port to join for a node"`
	Bootstrap   bool   `long:"bootstrap" env:"BOOTSTRAP" description:"bootstrap a cluster"`
	DataDir     string `long:"datadir" env:"DATA_DIR" default:"/tmp/data/" description:"Where to store system data"`
}

func main() {
	// We must read application options from command line
	// then initialize a raft node with this options.
	//
	// Also, the leader node must start a web interface
	// so other nodes will be able to join the cluster.
	// Basically, they send a POST request to the leader with their IP address,
	// and it adds them to the cluster
	opts := readOpts()

	log.Printf("[INFO] '%s' is used to store files of the node", opts.DataDir)

	config := node.Config{
		BindAddress:    opts.BindAddress,
		NodeIdentifier: opts.BindAddress,
		JoinAddress:    opts.JoinAddress,
		DataDir:        opts.DataDir,
		Bootstrap:      opts.Bootstrap,
	}
	storage, err := node.NewRStorage(&config)
	if err != nil {
		log.Panic(err)
	}

	msg := fmt.Sprintf("[INFO] Started node=%s", storage.RaftNode)
	log.Println(msg)

	go printStatus(storage)

	// If JoinAddress is not nil and tehre is no cluster, we have to send a POST request to this address
	// It must be an address of the cluster leader
	// We send POST request every second until it succeed
	servers, err := storage.GetClusterServers()
	notInCluster := (err != nil || len(servers) < 2)
	if config.JoinAddress != "" && notInCluster {
		for 1 == 1 {
			time.Sleep(time.Second * 1)
			err := joinCluster(storage, config.JoinAddress, config.BindAddress)
			if err != nil {
				log.Printf("[ERROR] Can't join the cluster: %+v", err)
			} else {
				break
			}
		}
	}

	// Start an HTTP server
	server.RunHTTPServer(storage)
}

func readOpts() Opts {
	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, err := p.ParseArgs(os.Args[1:]); err != nil {
		log.Panicln(err)
	}
	return opts
}

// joinCluster sends a POST request to "join" address
// to ask the cluster leader join this node as a voter
func joinCluster(storage *node.RStorage, address string, myAddress string) error {
	body, err := json.Marshal(map[string]string{"address": myAddress})
	if err != nil {
		return err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/cluster/join/", address),
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

func printStatus(storage *node.RStorage) {
	for 1 == 1 {
		log.Printf("[DEBUG] state=%s leader=%s", storage.RaftNode.State(), storage.RaftNode.Leader())
		time.Sleep(time.Second * 2)
	}
}
