package recursor

import (
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
