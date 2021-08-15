package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/filecoin-project/storetheindex/config"
	"github.com/filecoin-project/storetheindex/internal/providers/discovery"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

type mockDiscovery struct {
	discoverRsp *discovery.Discovered
}

const (
	exceptID  = "12D3KooWK7CTS7cyWi51PeNE3cTjS2F2kDCZaQVU4A5xBmb9J1do"
	trustedID = "12D3KooWSG3JuvEjRkSxt93ADTjQxqe4ExbBwSkQ9Zyk1WfBaZJF"

	minerDiscoAddr = "stitest999999"
	minerAddr      = "/ip4/127.0.0.1/tcp/9999"
)

var providersCfg = config.Providers{
	Policy:         "block",
	Except:         []string{exceptID},
	Trust:          []string{trustedID},
	PollInterval:   config.Duration(time.Minute),
	RediscoverWait: config.Duration(time.Minute),
}

func newMockDiscovery(t *testing.T, providerID string) *mockDiscovery {
	peerID, err := peer.Decode(providerID)
	if err != nil {
		t.Fatal("bad provider ID:", err)
	}

	maddr, err := multiaddr.NewMultiaddr(minerAddr)
	if err != nil {
		t.Fatal("bad miner address:", err)
	}

	return &mockDiscovery{
		discoverRsp: &discovery.Discovered{
			AddrInfo: peer.AddrInfo{
				ID:    peerID,
				Addrs: []multiaddr.Multiaddr{maddr},
			},
			Type: discovery.MinerType,
		},
	}
}

func (m *mockDiscovery) Discover(ctx context.Context, filecoinAddr string, signature, signed []byte) (*discovery.Discovered, error) {
	if filecoinAddr == "bad1234" {
		return nil, errors.New("unknown miner")
	}

	return m.discoverRsp, nil
}

func TestNewRegistryDiscovery(t *testing.T) {
	mockDisco := newMockDiscovery(t, exceptID)

	r, err := NewRegistry(providersCfg, nil, mockDisco)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("created new registry")

	peerID, err := peer.Decode(trustedID)
	if err != nil {
		t.Fatal("bad provider ID:", err)
	}
	info := &ProviderInfo{
		AddrInfo: peer.AddrInfo{
			ID: peerID,
		},
	}

	err = r.Register(info)
	if err != nil {
		t.Error("failed to register directly:", err)
	}

	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Check that 2nd call to Close is ok
	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDiscoveryAllowed(t *testing.T) {
	mockDisco := newMockDiscovery(t, exceptID)

	r, err := NewRegistry(providersCfg, nil, mockDisco)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	t.Log("created new registry")

	err = r.Discover(minerDiscoAddr, nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("discovered mock miner", minerDiscoAddr)

	info := r.ProviderInfoByAddr(minerDiscoAddr)
	if info == nil {
		t.Error("did not get provider info for miner")
	}
	t.Log("got provider info for miner")

	peerID, err := peer.Decode(exceptID)
	if err != nil {
		t.Fatal("bad provider ID:", err)
	}

	if info.AddrInfo.ID != peerID {
		t.Error("did not get correct porvider id")
	}

	peerID, err = peer.Decode(trustedID)
	if err != nil {
		t.Fatal("bad provider ID:", err)
	}
	info = &ProviderInfo{
		AddrInfo: peer.AddrInfo{
			ID: peerID,
		},
	}
	err = r.Register(info)
	if err != nil {
		t.Error("failed to register directly:", err)
	}

	infos := r.AllProviderInfo()
	if len(infos) != 2 {
		t.Fatal("expected 2 provider infos")
	}
}

func TestDiscoveryBlocked(t *testing.T) {
	providersCfg.Policy = "allow"
	defer func() {
		providersCfg.Policy = "block"
	}()

	mockDisco := newMockDiscovery(t, exceptID)

	r, err := NewRegistry(providersCfg, nil, mockDisco)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	t.Log("created new registry")

	err = r.Discover(minerDiscoAddr, nil, nil, true)
	if err != ErrNotAllowed {
		t.Fatal("expected error:", ErrNotAllowed)
	}

	into := r.ProviderInfoByAddr(minerDiscoAddr)
	if into != nil {
		t.Error("should not have found provider info for miner")
	}
}
