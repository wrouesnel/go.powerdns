package recursor

import (
	"reflect"

	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
)

// Kind is a string representing the type of zone in powerdns recursor
type Kind string

// nolint: golint
const (
	KindNative    Kind = "Native"
	KindForwarded Kind = "Forwarded"
)

// Zone implements the recusor nameserver zone subtype.
type Zone struct {
	shared.Zone
	Servers          []string `json:"servers"`
	RecursionDesired bool     `json:"recursion_desired"`
}

// HeaderEquals compares the Zone header metadata that would match between a ZoneRequest and a ZoneResponse.
// i.e. it does not compare RRsets or serials.
func (z *Zone) HeaderEquals(a Zone) bool {
	return z.Zone.HeaderEquals(a.Zone) &&
		reflect.DeepEqual(z.Servers, a.Servers) &&
		z.RecursionDesired == a.RecursionDesired
}

// Equals does a HeaderCompare and checks if the contained zones are exactly early
func (z *Zone) Equals(a Zone) bool {
	return z.Zone.HeaderEquals(a.Zone) && z.Zone.Equals(a.Zone)
}

// Copy makes a value based copy of the zone
func (z *Zone) Copy() Zone {
	r := Zone{}
	r.Zone = z.Zone.Copy()
	r.Servers = z.Servers[:]
	r.RecursionDesired = z.RecursionDesired
	return r
}
