package main

import (
	"context"
	"flag"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	api "github.com/osrg/gobgp/api"
	gobgp "github.com/osrg/gobgp/pkg/server"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
)

var (
	resumeFile      = flag.String("resume", "resume.txt", "path to resume file")
	localAsn        = flag.Uint("local-asn", 34553, "local ASN")
	localAddress    = flag.String("local-addr", "127.0.0.1", "local address to bind to")
	upstreamAsn     = flag.Uint("upstream-asn", 34553, "upstream's ASN")
	upstreamAddress = flag.String("upstream-addr", "127.0.0.2", "upstream's peering address")
	multihop        = flag.Bool("multihop", false, "enable BGP multihop")
)

func main() {
	flag.Parse()

	log.SetLevel(log.DebugLevel)

	// Create GoBGP server
	s := gobgp.NewBgpServer()
	go s.Serve()

	// global configuration
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         uint32(*localAsn),
			RouterId:   *localAddress,
			ListenPort: 179,
		},
	}); err != nil {
		log.Fatal(err)
	}

	// monitor the change of the peer state
	if err := s.MonitorPeer(context.Background(), &api.MonitorPeerRequest{}, func(p *api.Peer) { log.Info(p) }); err != nil {
		log.Fatal(err)
	}

	// neighbor configuration
	n := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: *upstreamAddress,
			PeerAs:          uint32(*upstreamAsn),
		},
	}

	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		log.Fatal(err)
	}

	// add routes

	v6Family := &api.Family{
		Afi:  api.Family_AFI_IP6,
		Safi: api.Family_SAFI_UNICAST,
	}

	// add v6 route
	nlri, _ = ptypes.MarshalAny(&api.IPAddressPrefix{
		PrefixLen: 64,
		Prefix:    "2001:db8:1::",
	})
	v6Attrs, _ := ptypes.MarshalAny(&api.MpReachNLRIAttribute{
		Family:   v6Family,
		NextHops: []string{"2001:db8::1"},
		Nlris:    []*any.Any{nlri},
	})

	c, _ := ptypes.MarshalAny(&api.CommunitiesAttribute{
		Communities: []uint32{100, 200},
	})

	_, err = s.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family: v6Family,
			Nlri:   nlri,
			Pattrs: []*any.Any{a1, v6Attrs, c},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	s.ListPath(context.Background(), &api.ListPathRequest{Family: v6Family}, func(p *api.Destination) {
		log.Info(p)
	})

	// Wait until interrupt
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)
	for range interruptChannel {
		log.Infoln("interrupted")
		os.Exit(0)
	}
}
