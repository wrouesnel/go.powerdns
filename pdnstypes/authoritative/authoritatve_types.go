package authoritative

import (
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
)

// Kind is a string representing the type of zone in powerdns authoritative
type Kind string

// nolint: golint
const (
	KindNative Kind = "Native"
	KindMaster Kind = "Master"
	KindSlave  Kind = "Slave"
)

// SoaEditValue should only ever be one of an SoaEditValue constant. The available constants are
// only those from the recommended set.
type SoaEditValue string

// nolint: golint
const (
	SoaEditValueIncrementWeeks     SoaEditValue = "INCREMENT-WEEKS"
	SoaEditValueInceptionEpoch     SoaEditValue = "INCEPTION-EPOCH"
	SoaEditValueInceptionIncrement SoaEditValue = "INCEPTION-INCREMENT"
	SoaEditValueNone               SoaEditValue = "NONE"
)

// RRsetChangeType is a fixed set of string constants used when patching zones.
type RRsetChangeType string

// nolint: golint
const (
	RRsetReplace RRsetChangeType = "REPLACE"
	RRSetDelete  RRsetChangeType = "DELETE"
)

// Zone implements the authoritative nameserver zone subtype.
type Zone struct {
	shared.Zone
	Kind   Kind `json:"kind"`
	DNSsec bool `json:"dnssec"`
	// The following are unimplemented as per the API spec
	//"nsec3param": "<nsec3param record>",
	//"nsec3narrow": <bool>,
	//"presigned": <bool>,
	SoaEdit    SoaEditValue `json:"soa_edit"`
	SoaEditAPI SoaEditValue `json:"soa_edit_api"`
	Account    string       `json:"account,omit_empty"`
}

// ZoneResponse implements the extra fields which are included in a response from a PowerDNS server. It should not
// be used to send a Zone request.
type ZoneResponse struct {
	Zone
	Serial         int `json:"serial"`
	NotifiedSerial int `json:"notified_serial"`
}

// ZoneRequestMaster implements the fields used when creating a master zone
type ZoneRequestMaster struct {
	Zone
	Nameservers []string `json:"nameservers"`
}

// ZoneRequestSlave implements the fields used when creating a slave zone
type ZoneRequestSlave struct {
	Zone
}

// ZoneRequestNative implements the fields used when creating a native zone
type ZoneRequestNative struct {
	Zone
	Nameservers []string `json:"nameservers"`
}

// RRsetPatchRequest implements the fields used when patching a specific RRset.
type RRsetPatchRequest struct {
	shared.RRset
	ChangeType RRsetChangeType `json:"changetype"`
}
