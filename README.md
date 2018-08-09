# Simple hashicorp/raft usage example

[![Build Status](https://travis-ci.org/alexander-akhmetov/raft-kv-example.svg?branch=master)](https://travis-ci.org/alexander-akhmetov/raft-kv-example)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexander-akhmetov/raft-kv-example)](https://goreportcard.com/report/github.com/alexander-akhmetov/raft-kv-example)

This is a simple distributed in-memory key-value storage which has an HTTP interface and uses [hashicorp/raft](https://github.com/hashicorp/raft) internally.

## Why?

I wanted to try to build something with Raft, so I decided to create yet another key-value storage. It will run elections to choose a new leader automatically if the current leader fails.
It's very simple in-memory key-value storage which is not meant to be a reliable KV DB and not for production usage of course :)

Useful links:

* [hashicorp/raft implementation](https://github.com/hashicorp/raft)
* [goraft/raft (unmaintained)](https://github.com/goraft/raft)
* [thesecretlivesofdata: raft explanation](http://thesecretlivesofdata.com/raft/)
* [raft.github.io](https://raft.github.io)

## Quick start

This project has configured docker containers ready to use. Start:

```bash
make run
```

It will start three docker containers (3 nodes) and you will see something like that

```none
...

node_2_1  | [GIN-debug] Listening and serving HTTP on :8080
node_1_1  | [raft][WARN] raft: AppendEntries to {Voter 10.1.0.102:4002 10.1.0.102:4002} rejected, sending older logs (next: 1)
node_2_1  | [raft][WARN] raft: Failed to get previous log: 4 log not found (last: 0)
node_1_1  | [raft][INFO] raft: pipelining replication to peer {Voter 10.1.0.102:4002 10.1.0.102:4002}
node_3_1  | [raft][DEBUG] raft-net: 10.1.0.103:4003 accepted connection from: 10.1.0.101:49686
node_2_1  | [raft][DEBUG] raft-net: 10.1.0.102:4002 accepted connection from: 10.1.0.101:39628
node_1_1  | [DEBUG] state=Leader leader=10.1.0.101:4001
node_3_1  | [DEBUG] state=Follower leader=10.1.0.101:4001
node_2_1  | [DEBUG] state=Follower leader=10.1.0.101:4001
node_1_1  | [DEBUG] state=Leader leader=10.1.0.101:4001

...
```

Now you can make requests to set and get keys:

```bash
########### Get value

~/ > curl 'http://127.0.0.1:4001/keys/some-key/'

{"value":""}  # empty, we don't have anything yet

########### Set value

~/ > curl 'http://127.0.0.1:4001/keys/some-key/' -H 'Content-Type: application/json' -d '{"value": "some-value"}'

{"value":"some-value"}  # saved

########### Get value again

~/ > curl 'http://127.0.0.1:4001/keys/some-key/'

{"value":"some-value"}  # hooray!
```

Try to stop the leader node, see how they elect a new leader and have fun! :)

## HTTP API description

API is very simple:

```none
POST /keys/<key>/

    Headers:
        Content-Type: application/jspn

    Request:

        {
            "value": "some-value"
        }

---------------------------------

GET /keys/<key>/

    Response:
        {
            "value": "some-value"
        }
```

## Docker

[docker-compose.yml](docker-compose.yml) file contains prepared cluster with three nodes. Basically, they are copies of the image from `Dockerfile`.
They will be exposed on ports `4001, 4002, 4003`.

## Example of the election

```bash
# we have a full cluster with three nodes in it: leader and two followers

node_1_1  | [DEBUG] state=Leader leader=10.1.0.100:4000
node_2_1  | [DEBUG] state=Follower leader=10.1.0.100:4000
node_3_1  | [DEBUG] state=Follower leader=10.1.0.100:4000

# let's stop the leader

> docker stop raftexample_node_1_1

raftexample_node_1_1 exited with code 137

# and the election process begins

node_3_1  | [raft][INFO] raft: Node at 10.1.0.103:4000 [Candidate] entering Candidate state in term 17
node_3_1  | [raft][INFO] raft: Election won. Tally: 2
node_3_1  | [raft][INFO] raft: Node at 10.1.0.103:4000 [Leader] entering Leader state
node_2_1  | [DEBUG] state=Follower leader=10.1.0.103:4000
node_3_1  | [DEBUG] state=Leader leader=10.1.0.103:4000

...

# now we have a new leader and one follower. Let's start old leader

> docker start raftexample_node_1_1

node_1_1  | [raft][DEBUG] raft-net: 10.1.0.100:4000 accepted connection from: 10.1.0.103:52494
node_1_1  | [DEBUG] state=Follower leader=10.1.0.103:4000
node_2_1  | [DEBUG] state=Follower leader=10.1.0.103:4000
node_3_1  | [DEBUG] state=Leader leader=10.1.0.103:4000

# old leader started and it has follower role now
```

## TODO

* More tests
* Ability to POST key to any node in the cluster, it will forward request to the leader automatically
* [Fuzzing Raft for Fun](https://colin-scott.github.io/blog/2015/10/07/fuzzing-raft-for-fun-and-profit/)
