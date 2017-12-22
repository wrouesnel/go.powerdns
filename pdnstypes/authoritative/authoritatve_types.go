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

// Equals compares the Zone header metadata that would match between a ZoneRequest and a ZoneResponse as well as
// the RRsets in the zone. It does not compare serials or notified serials.
func (z *Zone) Equals(a Zone) bool {
	return z.Zone.Equals(a.Zone) && z.HeaderEquals(a)
}

// Copy makes a value based copy of the zone
func (z *Zone) Copy() Zone {
	r := *z
	r.Zone = z.Zone.Copy()
	return r
}

// ZoneResponse implements the extra fields which are included in a response from a PowerDNS server. It should not
// be used to send a Zone request.
type ZoneResponse struct {
	Zone
	Serial         uint32 `json:"serial"`
	NotifiedSerial uint32 `json:"notified_serial"`
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

// PatchRRsets is a collection of PatchRRSet structs suitable for use with a patch request.
type PatchRRSets []PatchRRSet

// NewPatchRRSets initializes a new PatchRRSet from an RRSet, with the given changetype.
// The new RRset is initialized with copies of the original.
func NewPatchRRSets(rrsets shared.RRsets, changetype RRsetChangeType) PatchRRSets {
	result := make(PatchRRSets, 0, len(rrsets))
	for _, rrset := range rrsets {
		result = append(result, PatchRRSet{
			RRset:      rrset.Copy(),
			ChangeType: changetype,
		})
	}
	return result
}

// CopyToRRSets makes a value-based copy of all contained RRsets and returns a regular
// RRset object.
func (prrs PatchRRSets) CopyToRRSets() shared.RRsets {
	result := make(shared.RRsets, 0, len(prrs))
	for _, v := range prrs {
		result = append(result, v.CopyToRRSet())
	}
	return result
}

// PatchRRset is the RRSet type including the ChangeType field.
type PatchRRSet struct {
	shared.RRset
	ChangeType RRsetChangeType `json:"changetype"`
}

// CopyToRRSet makes a Copy of the contained RRset and returns it.
func (prrs *PatchRRSet) CopyToRRSet() shared.RRset {
	return prrs.RRset.Copy()
}

// PatchZoneRequest implements the fields used when creating a zone PATCH request
type PatchZoneRequest struct {
	RRSets PatchRRSets `json:"rrsets"`
}

// PatchZoneResponse implements the fields used when receiving the result of a successful zone PATCH request
type PatchZoneResponse ZoneResponse
