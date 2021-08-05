package command

import (
	"context"
	"errors"

	httpclient "github.com/filecoin-project/storetheindex/api/v0/client/http"
	"github.com/filecoin-project/storetheindex/server/net"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/urfave/cli/v2"
)

var importCidList = &cli.Command{
	Name:   "cidlist",
	Usage:  "Import indexer data from cidList",
	Flags:  ImportFlags,
	Action: importListCmd,
}

var importCar = &cli.Command{
	Name:   "car",
	Usage:  "Import indexer data from car",
	Flags:  ImportFlags,
	Action: importCarCmd,
}

var importManifest = &cli.Command{
	Name:   "manifest",
	Usage:  "Import manifest of CID aggregator",
	Flags:  ImportFlags,
	Action: importManifestCmd,
}
var ImportCmd = &cli.Command{
	Name:  "import",
	Usage: "Imports data to indexer from different sources",
	Subcommands: []*cli.Command{
		importCidList,
		importCar,
		importManifest,
	},
}

func importListCmd(cctx *cli.Context) error {
	// NOTE: Importing manually from CLI only supported for http protocol
	// for now. This feature is mainly for testing purposes
	endpoint := cctx.String("admin_ep")
	cl, err := httpclient.New()
	if err != nil {
		return err
	}
	end := net.NewHTTPEndpoint(endpoint)
	prov := cctx.String("provider")
	p, err := peer.Decode(prov)
	if err != nil {
		return err
	}
	dir := cctx.String("dir")

	log.Infow("Starting to import from cidlist file")
	// TODO: Should there be a timeout?  Since this may take a long time, it
	// would make sense that the request should complete immediately with a
	// redirect to a URL where the status can be polled for.
	return cl.ImportFromCidList(context.Background(), dir, p, end)
}

func importCarCmd(c *cli.Context) error {
	//log.Infow("Starting to import from CAR file")
	return errors.New("importing from car not implemented yet")
}

func importManifestCmd(cctx *cli.Context) error {
	endpoint := cctx.String("admin_ep")
	cl, err := httpclient.New()
	if err != nil {
		return err
	}
	end := net.NewHTTPEndpoint(endpoint)
	prov := cctx.String("provider")
	p, err := peer.Decode(prov)
	if err != nil {
		return err
	}
	dir := cctx.String("dir")

	log.Infow("Starting to import from manifest file")
	// TODO: Should there be a timeout?  Since this may take a long time, it
	// would make sense that the request should complete immediately with a
	// redirect to a URL where the status can be polled for.
	return cl.ImportFromManifest(context.Background(), dir, p, end)
}
