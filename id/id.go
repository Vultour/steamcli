// Package id provides a simple way to handle and convert various types of Steam IDs
package id

import (
	"fmt"
	"strconv"
	"strings"
)

// The following constants describe the Universe an ID can be attached to
const (
	UniverseUnspecified  = 0
	UniversePublic       = 1
	UniverseBeta         = 2
	UniverseInternal     = 3
	UniverseDev          = 4
	UniverseRC           = 5
	UniverseMaximumValue = 5
)

// The following constants describe the various account types
const (
	AccountInvalid        = 0
	AccountIndividual     = 1
	AccountMultiseat      = 2
	AccountGameServer     = 3
	AccountAnonGameServer = 4
	AccountPending        = 5
	AccountContentServer  = 6
	AccountClan           = 7
	AccountChat           = 8
	AccountP2PSuperSeeder = 9
	AccountAnonUser       = 10
	AccountMaximumValue   = 10
)

const (
	// CommunityBaseURL is the root of the Steam Community profile links
	CommunityBaseURL = "steamcommunity.com"
)

// CommunityPath describes the URL path for a given Account type
var CommunityPath = map[int]string{
	1: "profiles/id", // AccountIndividual
	7: "groups/id",   // AccountClan
}

// CommunityIdentifier is a special value used for generating a community URL
var CommunityIdentifier = map[int]int64{
	1: 0x0110000100000000, // AccountIndividual
	7: 0x0170000000000000, // AccountClan
}

// SteamIDFull is a completely decoded Steam ID
type SteamIDFull struct {
	Universe      uint64
	Type          uint64
	Instance      uint64
	AccountNumber uint64
	Y             uint64 // Should be a parity bit
}

// SteamIDTriplet describes the standard format of STEAM_X:Y:Z
type SteamIDTriplet struct {
	X int
	Y int // Should be a parity bit
	Z int
}

// SteamID contains all the transcoded versions of a given Steam ID
type SteamID struct {
	Full    SteamIDFull
	Triplet SteamIDTriplet
}

// ParseError is returned when parsing an ID fails
type ParseError struct {
	detail     string
	underlying string
}

// New attempts to parse the given string into a Steam ID
// This will be a networked operation if provided with a community ID
func New(id string) (*SteamID, error) {
	idd := strings.TrimSpace(id)
	if idd == "STEAM_ID_PENDING" {
		return &SteamID{}, &ParseError{detail: "Cannot parse PENDING triplets"}
	}

	if idd == "UNKNOWN" {
		return &SteamID{}, &ParseError{detail: "This ID is invalid (UNKNOWN)"}
	}

	if id64, err := strconv.ParseUint(idd, 10, 64); err == nil {
		idFull := SteamIDFull{
			Y:             id64 & 0x1,
			AccountNumber: id64 & 0xFFFFFFE,
			Instance:      id64 & 0xFFFFF00000000,
			Type:          id64 & 0xF0000000000000,
			Universe:      id64 & 0xFF00000000000000,
		}

		if !(idFull.Type > AccountMaximumValue) &&
			!(idFull.Universe > UniverseMaximumValue) {
			return &SteamID{
				Full: idFull,
				Triplet: SteamIDTriplet{
					X: int(idFull.Universe),
					Y: int(idFull.Y),
					Z: int(idFull.AccountNumber),
				},
			}, nil
		}
	}

	if strings.HasPrefix(idd, "STEAM_") {
		data := strings.Split(id[6:], ":")
		if len(data) != 3 {
			return &SteamID{}, &ParseError{
				detail:     "Could not parse triplet",
				underlying: "ID doesn't contain three parts",
			}
		}

		x, err := strconv.ParseInt(data[0], 10, 32)
		if err != nil {
			return &SteamID{}, &ParseError{
				detail:     "Could not parse triplet value X",
				underlying: err.Error(),
			}
		}
		y, err := strconv.ParseInt(data[1], 10, 32)
		if err != nil {
			return &SteamID{}, &ParseError{
				detail:     "Could not parse triplet value Y",
				underlying: err.Error(),
			}
		}
		z, err := strconv.ParseInt(data[2], 10, 32)
		if err != nil {
			return &SteamID{}, &ParseError{
				detail:     "Could not parse triplet value Z",
				underlying: err.Error(),
			}
		}

		triplet := SteamIDTriplet{
			X: int(x),
			Y: int(y),
			Z: int(z),
		}

		// Guessing type and instance!
		return &SteamID{
			Triplet: triplet,
			Full: SteamIDFull{
				AccountNumber: uint64(triplet.Z),
				Y:             uint64(triplet.Y),
				Universe:      uint64(triplet.X),
				Instance:      1,
				Type:          1,
			},
		}, nil
	}

	return &SteamID{}, &ParseError{detail: "Couldn't determine ID type"}
}

// CommunityID returns the ID used for generating a community profile URL
func (id *SteamID) CommunityID() int64 {
	V, _ := CommunityIdentifier[int(id.Full.Type)]
	return int64(id.Triplet.Z*2) + int64(V) + int64(id.Triplet.Y)
}

// CommunityURL returns a link to the ID's profile (without protocol)
func (id *SteamID) CommunityURL() string {
	return fmt.Sprintf(
		"%s/%s/%d",
		CommunityBaseURL,
		CommunityPath[int(id.Full.Type)],
		id.CommunityID(),
	)
}

// Get returns the 64bit SteamID (this is _not_ the Community ID!)
func (id *SteamIDFull) Get() uint64 {
	var ret uint64
	ret = (ret | (id.Y << 63)) >> 31
	ret = (ret | (id.AccountNumber << 33)) >> 20
	ret = (ret | (id.Instance << 44)) >> 4
	ret = (ret | (id.Type << 60)) >> 8
	ret = (ret | (id.Universe << 56))
	return ret
}

func (e *ParseError) Error() string {
	if e.underlying == "" {
		e.underlying = "none"
	}
	return fmt.Sprintf(
		"ParseError: %s (underlying: %s)",
		e.detail,
		e.underlying,
	)
}

func (id *SteamID) String() string {
	return fmt.Sprintf(
		`Steam ID:
	Triplet: STEAM_%d:%d:%d
		X: %d
		Y: %d
		Z: %d
	Full: %d
		Account Number: %d
		Universe:       %d
		Y:              %d
		Instance:       %d
		Type:           %d
	Community:
		ID: %d
		URL: %s`,
		id.Triplet.X, id.Triplet.Y, id.Triplet.Z,
		id.Triplet.X, id.Triplet.Y, id.Triplet.Z,
		id.Full.Get(),
		id.Full.AccountNumber, id.Full.Universe, id.Full.Y, id.Full.Instance,
		id.Full.Type,
		id.CommunityID(),
		id.CommunityURL(),
	)
}
