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
	Account    string       `json:"account,omitempty"`
}

// HeaderEquals compares the Zone header metadata that would match between a ZoneRequest and a ZoneResponse.
// i.e. it does not compare RRsets or serials.
func (z *Zone) HeaderEquals(a Zone) bool {
	return z.Zone.HeaderEquals(a.Zone) &&
		z.Kind == a.Kind &&
		z.DNSsec == a.DNSsec &&
		z.SoaEdit == a.SoaEdit &&
		z.SoaEditAPI == a.SoaEditAPI &&
		z.Account == a.Account
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

// PatchZoneRequest implements the fields used when creating a zone PATCH request
type PatchZoneRequest struct {
	RRSets shared.RRsets `json:"rrsets"`
}

// PatchZoneResponse implements the fields used when receiving the result of a successful zone PATCH request
type PatchZoneResponse ZoneResponse
