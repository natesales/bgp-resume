package encoding

import (
	"fmt"
	"strconv"
	"strings"

	api "github.com/osrg/gobgp/api"
	log "github.com/sirupsen/logrus"
)

// Marshal converts a string into a list of BGP large communities
func Marshal(input string, asn uint32) []*api.LargeCommunity {
	// Convert input string to list of 3 digit values
	var paddedCharacters [][]string
	for _, letter := range input {
		asciiCode := int(letter) // Get the rune's ASCII code

		// Pad small numbers
		var paddedChar string
		if asciiCode < 10 {
			paddedChar = "00" + strconv.Itoa(asciiCode)
		} else if asciiCode < 100 {
			paddedChar = "0" + strconv.Itoa(asciiCode)
		} else { // If the ASCII code is 3 digits, don't pad it
			paddedChar = strconv.Itoa(asciiCode)
		}

		// If this is the first element, create a new slice
		if paddedCharacters == nil {
			paddedCharacters = [][]string{{paddedChar}}
		} else {
			// For all subsequent elements, check if the array is already 3 items in length
			if len(paddedCharacters[len(paddedCharacters)-1]) < 3 {
				paddedCharacters[len(paddedCharacters)-1] = append(paddedCharacters[len(paddedCharacters)-1], paddedChar)
			} else {
				// If the last a
				paddedCharacters = append(paddedCharacters, []string{paddedChar})
			}
		}
	}

	// Add null array elements until the last array is 3 items long
	for range paddedCharacters {
		if len(paddedCharacters[len(paddedCharacters)-1]) != 3 {
			paddedCharacters[len(paddedCharacters)-1] = append(paddedCharacters[len(paddedCharacters)-1], "000")
		}
	}

	// Concatenate each array element into a uint32
	var localData []uint32
	for _, elem := range paddedCharacters {
		elemString := "1" + strings.Join(elem, "")
		elemUint32, err := strconv.Atoi(elemString)
		if err != nil {
			log.Fatalf("protocol error: unable to convert %s to a integer", elemString)
		}

		localData = append(localData, uint32(elemUint32))
	}

	fmt.Println(localData)

	fmt.Println("Padded characters:")
	fmt.Println(paddedCharacters)
	fmt.Println("Padded characters length:")
	fmt.Println(len(paddedCharacters))

	// Assemble the integers into communities
	var communities []*api.LargeCommunity
	for i := 0; i < len(localData); i += 2 {
		if i < len(localData)-2 {
			communities = append(communities, &api.LargeCommunity{
				GlobalAdmin: asn,
				LocalData1:  localData[i],
				LocalData2:  localData[i+1],
			})
		} else {
			communities = append(communities, &api.LargeCommunity{
				GlobalAdmin: asn,
				LocalData1:  localData[i],
				LocalData2:  1000000000,
			})
		}
	}

	log.Infof("Converted %d bytes into %d communities\n", len(input), len(communities))

	return communities
}

// toCharacter converts a 3 rune string into a single ASCII character
func toCharacter(input string) string {
	// Ignore null entries
	if input == "000" {
		return ""
	}

	// Trim the leading zero pad
	var partialValue string
	if input[:1] == "0" {
		partialValue = input[1:]
	} else {
		partialValue = input
	}

	// Convert integer to letter
	strAsInt, _ := strconv.Atoi(partialValue)
	return string(rune(strAsInt))
}

// Unmarshal converts a list of BGP large communities into a string
func Unmarshal(communities []api.LargeCommunity, asn uint32) string {
	// Convert communities into list of integers
	var dataIntegers []uint32
	for _, community := range communities {
		if community.GlobalAdmin == asn {
			dataIntegers = append(dataIntegers, community.LocalData1)
			dataIntegers = append(dataIntegers, community.LocalData2)
		}
	}

	fmt.Println(dataIntegers)

	// Convert integers into characters
	output := ""
	for _, part := range dataIntegers {
		partString := strconv.Itoa(int(part))
		part1 := toCharacter(partString[1:4])
		part2 := toCharacter(partString[4:7])
		part3 := toCharacter(partString[7:10])
		output += part1 + part2 + part3
	}

	return output
}
