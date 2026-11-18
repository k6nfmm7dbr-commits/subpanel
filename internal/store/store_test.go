package store

import (
	"testing"

	"subpanel/internal/model"
)

func TestStoreRoundTrip(t *testing.T) {
	path := t.TempDir() + "/nodes.json"
	s, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	set := &NodeSet{Token: "tok1", Name: "g", Nodes: []model.Node{
		{Name: "A", Type: "ss", Server: "1.2.3.4", Port: 8388, Cipher: "aes-256-gcm", Password: "p"},
	}}
	if err := s.Upsert(set); err != nil {
		t.Fatal(err)
	}
	s2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.Get("tok1")
	if !ok || len(got.Nodes) != 1 {
		t.Fatalf("round trip failed: %+v", got)
	}
	if got.Nodes[0].ID == "" {
		t.Fatal("node ID not normalized")
	}
}
