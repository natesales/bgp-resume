package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	api "github.com/osrg/gobgp/api"
	log "github.com/sirupsen/logrus"

	"github.com/natesales/bgp-resume/internal/encoding"
)

var (
	format    = flag.String("format", "", "input router community format (bird)")
	asnFilter = flag.Int("asn", 0, "ASN to filter by")
)

func main() {
	flag.Parse()

	// Check for empty flags
	if *asnFilter == 0 || *format == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Read from stdin
	bytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	// List of received communities
	var communities []*api.LargeCommunity

	switch *format {
	case "bird":
		// Split by BGP large community attributes
		communitiesString := strings.Split(string(bytes), "BGP.large_community: ")
		if len(communitiesString) < 2 {
			log.Fatal("no communities found for route")
		}

		// Loop over communities, trimming parenthesis
		for _, community := range strings.Split(strings.ReplaceAll(communitiesString[1], ")", ""), "(") {
			parts := strings.Split(strings.TrimSpace(community), ", ")
			if len(parts) == 3 { // Only include lines that have all 3 parts of BGP large community
				asn, err := strconv.Atoi(parts[0])
				if err != nil {
					log.Fatalf("unable to parse ASN %d to an integer", asn)
				}

				// Ignore communities that don't start with the supplied ASN
				if asn != *asnFilter {
					continue
				}

				data1, err := strconv.Atoi(parts[1])
				if err != nil {
					log.Fatalf("unable to parse data %d to an integer", data1)
				}

				data2, err := strconv.Atoi(parts[2])
				if err != nil {
					log.Fatalf("unable to parse data %d to an integer", data2)
				}

				// Create and append a new community
				communities = append(communities, &api.LargeCommunity{
					GlobalAdmin: uint32(asn),
					LocalData1:  uint32(data1),
					LocalData2:  uint32(data2),
				})
			}
		}
	default: // If the supplied format doesn't have a parser, show the flags
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Decode the communities into a string
	outputString := encoding.Unmarshal(communities, uint32(*asnFilter))

	// Decode as base64
	decoded, _ := base64.StdEncoding.DecodeString(outputString)
	fmt.Println(string(decoded))
}
