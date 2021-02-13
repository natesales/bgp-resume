package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	api "github.com/osrg/gobgp/api"
	gobgp "github.com/osrg/gobgp/pkg/server"
	log "github.com/sirupsen/logrus"
)

var (
	resumeFile      = flag.String("resume", "", "path to resume file")
	localAsn        = flag.Uint("local-asn", 0, "local ASN")
	localAddress    = flag.String("local-addr", "", "local address to bind to")
	localPort       = flag.Uint("local-port", 179, "local BGP listen port")
	upstreamAsn     = flag.Uint("upstream-asn", 0, "upstream's ASN")
	upstreamAddress = flag.String("upstream-addr", "", "upstream's peering address")
	multihop        = flag.Bool("multihop", false, "enable BGP multihop")
	routerId        = flag.String("router-id", "", "BGP router id")
)

func main() {
	flag.Parse()

	// Validate flags
	if *resumeFile == "" || *localAsn == 0 || *localAddress == "" || *upstreamAsn == 0 || *upstreamAddress == "" || *routerId == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.SetLevel(log.DebugLevel)
	s := gobgp.NewBgpServer()
	go s.Serve()

	// global configuration
	if err := s.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         uint32(*localAsn),
			RouterId:   *routerId,
			ListenPort: int32(*localPort), // gobgp won't listen on tcp:179
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
	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    "10.0.0.0",
		PrefixLen: 24,
	})

	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: *localAddress,
	})
	a3, _ := ptypes.MarshalAny(&api.AsPathAttribute{
		Segments: []*api.AsSegment{
			{
				Type:    2,
				Numbers: []uint32{6762, 39919, 65000, 35753, 65000},
			},
		},
	})
	//communities, _ := ptypes.MarshalAny(&api.CommunitiesAttribute{
	//	Communities: []uint32{100, 200},
	//})
	attrs := []*any.Any{a1, a2, a3}

	_, err := s.AddPath(context.Background(), &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
			Nlri:   nlri,
			Pattrs: attrs,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// do something useful here instead of exiting
	time.Sleep(time.Minute * 3)
}
