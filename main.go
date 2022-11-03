package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"

	"raft-kv/node"
	"raft-kv/server"
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
	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, err := p.ParseArgs(os.Args[1:]); err != nil {
		log.Panicln(err)
	}

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

	// If JoinAddress is not nil and there is no cluster, we have to send a POST request to this address
	// It must be an address of the cluster leader
	// We send POST request every second until it succeed
	if config.JoinAddress != "" {
		for 1 == 1 {
			time.Sleep(time.Second * 1)
			err := storage.JoinCluster(config.JoinAddress)
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

func printStatus(s *node.RStorage) {
	for 1 == 1 {
		log.Printf("[DEBUG] state=%s leader=%s", s.RaftNode.State(), s.RaftNode.Leader())
		time.Sleep(time.Second * 2)
	}
}
