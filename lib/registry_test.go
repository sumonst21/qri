package lib

import (
	"context"
	"testing"

	testPeers "github.com/qri-io/qri/config/test"
	"github.com/qri-io/qri/profile"
	"github.com/qri-io/qri/registry/regserver"
	repotest "github.com/qri-io/qri/repo/test"
)

// Test that running prove sets the profileID for the user
func TestProveProfileKey(t *testing.T) {
	tr := newTestRunner(t)
	defer tr.Delete()

	ctx, cancel := context.WithCancel(context.Background())
	reg, cleanup, err := regserver.NewTempRegistry(ctx, "temp_registry", "", repotest.NewTestCrypto())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	defer cancel()

	// Create a mock registry, point our test runner to its URL
	regClient, _ := regserver.NewMockServerRegistry(*reg)
	tr.Instance.registry = regClient

	// Get an example peer, and add it to the local profile store
	info := testPeers.GetTestPeerInfo(2)
	pro := &profile.Profile{
		Peername: "test_peer",
		PubKey:   info.PubKey,
		PrivKey:  info.PrivKey,
		ID:       profile.IDFromPeerID(info.PeerID),
	}
	repo := tr.Instance.Repo()
	pstore := repo.Profiles()
	err = pstore.SetOwner(pro)
	if err != nil {
		t.Fatal(err)
	}

	// Call the endpoint to prove our account
	methods := NewRegistryClientMethods(tr.Instance)
	p := RegistryProfile{
		Username: pro.Peername,
		Email:    "test_peer@qri.io",
		Password: "hunter2",
	}
	ok := false
	err = methods.ProveProfileKey(&p, &ok)
	if err != nil {
		t.Error(err)
	}

	// Peer 3 is used by the mock regserver, it is now used by this peer
	expectProfileID := profile.IDFromPeerID(testPeers.GetTestPeerInfo(3).PeerID)
	if pro.ID != expectProfileID {
		t.Errorf("bad profileID for peer after prove. expect: %s, got: %s", expectProfileID, pro.ID)
	}
}
