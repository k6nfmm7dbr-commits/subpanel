package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// NodeSet is a collection of nodes behind one subscription token.
type NodeSet struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	Nodes []Node `json:"nodes"`
}

// Store persists node sets to a JSON file with atomic writes.
type Store struct {
	mu   sync.RWMutex
	path string
	sets map[string]*NodeSet
}

func OpenStore(path string) (*Store, error) {
	s := &Store{path: path, sets: make(map[string]*NodeSet)}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		var sets []*NodeSet
		if err := json.Unmarshal(b, &sets); err != nil {
			return nil, err
		}
		for _, set := range sets {
			s.sets[set.Token] = set
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return s, nil
}

func (s *Store) saveLocked() error {
	sets := make([]*NodeSet, 0, len(s.sets))
	for _, set := range s.sets {
		sets = append(sets, set)
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].Token < sets[j].Token })
	b, err := json.MarshalIndent(sets, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) List() []*NodeSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*NodeSet, 0, len(s.sets))
	for _, set := range s.sets {
		copySet := *set
		copySet.Nodes = append([]Node(nil), set.Nodes...)
		out = append(out, &copySet)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Token < out[j].Token })
	return out
}

func (s *Store) Get(token string) (*NodeSet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set, ok := s.sets[token]
	if !ok {
		return nil, false
	}
	copySet := *set
	copySet.Nodes = append([]Node(nil), set.Nodes...)
	return &copySet, true
}

func (s *Store) Upsert(set *NodeSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if set.Token == "" {
		set.Token = randomToken()
	}
	for i := range set.Nodes {
		normalizeNode(&set.Nodes[i])
	}
	s.sets[set.Token] = set
	return s.saveLocked()
}

func (s *Store) Delete(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sets, token)
	return s.saveLocked()
}
