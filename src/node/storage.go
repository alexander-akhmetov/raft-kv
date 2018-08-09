package node

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/raft"
)

// RStorage represents key-value storage with raft based replication
// Also, it represents finite-state machine which processes Raft log events
// https://godoc.org/github.com/hashicorp/raft#FSM
type RStorage struct {
	mutex    sync.Mutex
	storage  map[string]string
	RaftNode *raft.Raft
	config   Config
}

// Get value by key
func (s *RStorage) Get(key string) string {
	return s.storage[key]
}

// Set value by key
func (s *RStorage) Set(key string, value string) error {
	if s.RaftNode.State() != raft.Leader {
		return fmt.Errorf("Only leader can write to the storage")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	event := &logEvent{
		Type:  "set",
		Key:   key,
		Value: value,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	timeout := time.Second * 5
	_ = s.RaftNode.Apply(data, timeout)

	return nil
}

type logEvent struct {
	Type  string
	Key   string
	Value string
}

// Apply applies a Raft log entry to the key-value store.
func (s *RStorage) Apply(logEntry *raft.Log) interface{} {
	log.Println("[DEBUG] Applying a new log entry to the store")

	var event logEvent
	if err := json.Unmarshal(logEntry.Data, &event); err != nil {
		log.Println("[ERROR] Can't read Raft log event")
	}

	if event.Type == "set" {
		log.Printf("[DEBUG] set operation received key=%s value=%s", event.Key, event.Value)
		s.mutex.Lock()
		defer s.mutex.Unlock()
		s.storage[event.Key] = event.Value
		return nil
	}

	log.Printf("Unknown Raft log event type: %s", event.Type)
	return nil
}

// fsmSnapshot is used by Raft library to save a point-in-time snapshot of the FSM
// https://godoc.org/github.com/hashicorp/raft#FSMSnapshot
type fsmSnapshot struct {
	storage map[string]string
}

// Snapshot returns FSMSnapshot which is used to save snapshot of the FSM
func (s *RStorage) Snapshot() (raft.FSMSnapshot, error) {
	log.Println("[DEBUG] Snapshot")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	storageCopy := map[string]string{}
	for k, v := range s.storage {
		storageCopy[k] = v
	}

	return &fsmSnapshot{storage: storageCopy}, nil
}

// Restore stores the key-value store to a previous state.
func (s *RStorage) Restore(serialized io.ReadCloser) error {
	log.Println("[DEBUG] Restore")
	var snapshot fsmSnapshot
	if err := json.NewDecoder(serialized).Decode(&snapshot); err != nil {
		return err
	}

	s.storage = snapshot.storage
	return nil
}

// Persist should dump all necessary state to the WriteCloser 'sink',
// and call sink.Close() when finished or call sink.Cancel() on error.
// https://godoc.org/github.com/hashicorp/raft#FSMSnapshot
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	log.Println("[DEBUG] Persist")

	// trying to save a snapshot
	err := func() error {
		snapshotBytes, err := json.Marshal(f)
		if err != nil {
			return err
		}

		if _, err := sink.Write(snapshotBytes); err != nil {
			return err
		}

		err = sink.Close()
		if err != nil {
			return err
		}

		return nil
	}()

	// if it fails, we must call Cancel method to indicate unsuccessful end of the snapshoting process
	if err != nil {
		sink.Cancel()
		return err
	}

	return nil
}

// Release is invoked when the Raft library is finished with the snapshot.
// https://godoc.org/github.com/hashicorp/raft#FSMSnapshot
func (f *fsmSnapshot) Release() {
	log.Println("[DEBUG] Release")
}
