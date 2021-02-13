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
	"strconv"
	"strings"
)

var (
	resumeFile      = flag.String("resume", "", "path to resume file")
	asn             = flag.Uint("asn", 0, "ASN")
	localAddress    = flag.String("local-addr", "", "local address to bind to")
	upstreamAddress = flag.String("upstream-addr", "", "upstream's peering address")
	multihop        = flag.Bool("multihop", false, "enable BGP multihop")
	routerId        = flag.String("router-id", "", "BGP router id")
	prefix          = flag.String("prefix", "", "IPv6 prefix to announce")
)

func main() {
	flag.Parse()

	// Validate flags
	if *resumeFile == "" || *asn == 0 || *localAddress == "" || *upstreamAddress == "" || *routerId == "" || *prefix == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Parse prefix into network and mask
	prefixParts := strings.Split(*prefix, "/")
	prefixNetwork := prefixParts[0]
	prefixMask, err := strconv.Atoi(prefixParts[1])
	if err != nil {
		log.Fatalf("unable to parse prefix mask: %v", err)
	}

	// TODO: add startup info about what's going on

	log.SetLevel(log.DebugLevel)
	s := gobgp.NewBgpServer()
	go s.Serve()

	// global configuration
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         uint32(*asn),
			RouterId:   *routerId,
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
			PeerAs:          uint32(*asn),
		},
	}

	if err := s.AddPeer(context.Background(), &api.AddPeerRequest{
		Peer: n,
	}); err != nil {
		log.Fatal(err)
	}

	v6Family := &api.Family{
		Afi:  api.Family_AFI_IP6,
		Safi: api.Family_SAFI_UNICAST,
	}

	// BGP route attributes
	// TODO: error handling instead of ignoring
	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    prefixNetwork,
		PrefixLen: uint32(prefixMask),
	})

	originAttr, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})

	reachabilityAttrs, _ := ptypes.MarshalAny(&api.MpReachNLRIAttribute{
		Family:   v6Family,
		NextHops: []string{*localAddress},
		Nlris:    []*any.Any{nlri},
	})

	largeCommunities, _ := ptypes.MarshalAny(&api.LargeCommunitiesAttribute{
		Communities: []*api.LargeCommunity{},
	})

	pathAttrs := []*any.Any{originAttr, reachabilityAttrs, largeCommunities}

	_, err = s.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{Afi: api.Family_AFI_IP6, Safi: api.Family_SAFI_UNICAST},
			Nlri:   nlri,
			Pattrs: pathAttrs,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Wait until interrupt
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt)
	for range interruptChannel {
		log.Infoln("interrupted")
		os.Exit(0)
	}
}
