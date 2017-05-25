package recursor

import (
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
)

type Kind string
const (
	KindNative Kind = "Native"
	KindForwarded Kind = "Forwarded"
)

// Zone implements the recusor nameserver zone subtype.
type Zone struct {
	shared.Zone
	Servers []string `json:"servers"`
	RecursionDesired bool `json:"recursion_desired"`
}