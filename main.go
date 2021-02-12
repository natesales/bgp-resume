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
	resumeFile      = flag.String("resume", "", "path to resume file")
	localAsn        = flag.Uint("local-asn", 0, "local ASN")
	localAddress    = flag.String("local-addr", "", "local address to bind to")
	localPort       = flag.Uint("local-port", 179, "local BGP listen port")
	upstreamAsn     = flag.Uint("upstream-asn", 0, "upstream's ASN")
	upstreamAddress = flag.String("upstream-addr", "", "upstream's peering address")
	multihop        = flag.Bool("multihop", false, "enable BGP multihop")
)

func main() {
	flag.Parse()

	// Validate flags
	if *resumeFile == "" || *localAsn == 0 || *localAddress == "" || *upstreamAsn == 0 || *upstreamAddress == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetLevel(log.DebugLevel)

	// Create GoBGP server
	s := gobgp.NewBgpServer()
	go s.Serve()

	// Setup BGP listener
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         uint32(*localAsn),
			RouterId:   *localAddress,
			ListenPort: int32(*localPort),
		},
	}); err != nil {
		log.Fatal(err)
	}

	// Monitor peer state change
	if err := s.MonitorPeer(context.Background(), &api.MonitorPeerRequest{}, func(p *api.Peer) { log.Info(p) }); err != nil {
		log.Fatal(err)
	}

	// Configure the upstream BGP session
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

	// Add route
	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: uint32(*localAsn),
	})

	v6Family := &api.Family{
		Afi:  api.Family_AFI_IP6,
		Safi: api.Family_SAFI_UNICAST,
	}

	// add v6 route
	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
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

	_, err := s.AddPath(context.Background(), &api.AddPathRequest{
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
