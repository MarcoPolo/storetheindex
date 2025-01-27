package test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"
	"time"

	"github.com/filecoin-project/go-indexer-core"
	"github.com/filecoin-project/go-indexer-core/cache"
	"github.com/filecoin-project/go-indexer-core/cache/radixcache"
	"github.com/filecoin-project/go-indexer-core/engine"
	"github.com/filecoin-project/go-indexer-core/store/storethehash"
	v0 "github.com/filecoin-project/storetheindex/api/v0"
	"github.com/filecoin-project/storetheindex/api/v0/finder/client"
	"github.com/filecoin-project/storetheindex/api/v0/finder/model"
	"github.com/filecoin-project/storetheindex/config"
	"github.com/filecoin-project/storetheindex/internal/registry"
	"github.com/filecoin-project/storetheindex/test/util"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

const (
	providerID = "12D3KooWKRyzVWW6ChFjQjK4miCty85Niy48tpPV95XdKu1BcvMA"
	protocolID = 0x300000
)

//InitIndex initialize a new indexer engine.
func InitIndex(t *testing.T, withCache bool) indexer.Interface {
	var err error
	var tmpDir string
	if runtime.GOOS == "windows" {
		tmpDir, err = ioutil.TempDir("", "sth_test")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		tmpDir = t.TempDir()
	}
	valueStore, err := storethehash.New(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	var resultCache cache.Interface
	if withCache {
		resultCache = radixcache.New(100000)
	}
	return engine.New(resultCache, valueStore)
}

// InitRegistry initializes a new registry
func InitRegistry(t *testing.T) *registry.Registry {
	var discoveryCfg = config.Discovery{
		Policy: config.Policy{
			Allow:       false,
			Except:      []string{providerID},
			Trust:       false,
			TrustExcept: []string{providerID},
		},
		PollInterval:   config.Duration(time.Minute),
		RediscoverWait: config.Duration(time.Minute),
	}
	reg, err := registry.NewRegistry(discoveryCfg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// populateIndex with some multihashes
func populateIndex(ind indexer.Interface, mhs []multihash.Multihash, v indexer.Value, t *testing.T) {
	err := ind.Put(v, mhs...)
	if err != nil {
		t.Fatal("Error putting multihashes: ", err)
	}
	vals, ok, err := ind.Get(mhs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("index not found")
	}
	if len(vals) == 0 {
		t.Fatal("no values returned")
	}
	if !v.Equal(vals[0]) {
		t.Fatal("stored and retrieved values are different")
	}
}

func FindIndexTest(ctx context.Context, t *testing.T, c client.Finder, ind indexer.Interface, reg *registry.Registry) {
	// Generate some multihashes and populate indexer
	mhs := util.RandomMultihashes(15)
	p, err := peer.Decode(providerID)
	if err != nil {
		t.Fatal(err)
	}
	ctxID := []byte("test-context-id")
	metadata := v0.Metadata{
		ProtocolID: protocolID,
		Data:       []byte(mhs[0]),
	}
	encMetadata, err := metadata.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	v := indexer.Value{
		ProviderID:    p,
		ContextID:     ctxID,
		MetadataBytes: encMetadata,
	}
	populateIndex(ind, mhs[:10], v, t)

	a, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/9999")
	info := &registry.ProviderInfo{
		AddrInfo: peer.AddrInfo{
			ID:    p,
			Addrs: []multiaddr.Multiaddr{a},
		},
	}
	err = reg.Register(info)
	if err != nil {
		t.Fatal("could not register provider info:", err)
	}

	// Get single multihash
	resp, err := c.Find(ctx, mhs[0])
	if err != nil {
		t.Fatal(err)
	}
	t.Log("index values in resp:", len(resp.MultihashResults))

	provResult := model.ProviderResult{
		ContextID: v.ContextID,
		Provider: peer.AddrInfo{
			ID:    v.ProviderID,
			Addrs: info.AddrInfo.Addrs,
		},
	}
	err = provResult.Metadata.UnmarshalBinary(v.MetadataBytes)
	if err != nil {
		t.Fatal(err)
	}

	expectedResults := []model.ProviderResult{provResult}
	err = checkResponse(resp, mhs[:1], expectedResults)
	if err != nil {
		t.Fatal(err)
	}

	// Get a batch of multihashes
	resp, err = c.FindBatch(ctx, mhs[:10])
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(resp, mhs[:10], expectedResults)
	if err != nil {
		t.Fatal(err)
	}

	// Get a batch of multihashes where only a subset is in the index
	resp, err = c.FindBatch(ctx, mhs)
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(resp, mhs[:10], expectedResults)
	if err != nil {
		t.Fatal(err)
	}

	// Get empty batch
	_, err = c.FindBatch(ctx, []multihash.Multihash{})
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(&model.FindResponse{}, []multihash.Multihash{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Get batch with no multihashes in request
	_, err = c.FindBatch(ctx, mhs[10:])
	if err != nil {
		t.Fatal(err)
	}
	err = checkResponse(&model.FindResponse{}, []multihash.Multihash{}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

func checkResponse(r *model.FindResponse, mhs []multihash.Multihash, expected []model.ProviderResult) error {
	// Check if everything was returned.
	if len(r.MultihashResults) != len(mhs) {
		return fmt.Errorf("number of values send in responses not correct, expected %d got %d", len(mhs), len(r.MultihashResults))
	}
	for i := range r.MultihashResults {
		// Check if multihash in list of multihashes
		if !hasMultihash(mhs, r.MultihashResults[i].Multihash) {
			return fmt.Errorf("multihash not found in response containing %d multihash", len(mhs))
		}

		// Check if same value
		for j, pr := range r.MultihashResults[i].ProviderResults {
			if !pr.Equal(expected[j]) {
				return fmt.Errorf("wrong ProviderResult included for a multihash: %s", expected[j])
			}
		}
	}
	return nil
}

func hasMultihash(mhs []multihash.Multihash, m multihash.Multihash) bool {
	for i := range mhs {
		if bytes.Equal([]byte(mhs[i]), []byte(m)) {
			return true
		}
	}
	return false
}
