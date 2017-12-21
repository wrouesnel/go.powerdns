package shared

import (
	"fmt"
	"time"
)

// Error struct
type Error struct {
	Message string  `json:"error"`
	Errors  []Error `json:"errors,omitempty"`
}

// Error Returns
func (e Error) Error() string {
	return fmt.Sprintf("%v", e.Message)
}

// WrappedErrors implements errwrap.Wrapper
func (e Error) WrappedErrors() []error {
	ret := []error{}
	for _, err := range e.Errors {
		ret = append(ret, error(err))
	}
	return ret
}

// APIVersion struct
type APIVersion struct {
	URL     string `json:"url"`
	Version int    `json:"version"`
}

// DaemonType indicates the type of server in use.
type DaemonType string

// nolint: golint
const (
	DaemonTypeAuthoritative = "authoritative"
	DaemonTypeRecursor      = "recursor"
)

// ServerInfo struct
type ServerInfo struct {
	ConfigURL  string     `json:"config_url"`
	DaemonType DaemonType `json:"daemon_type"`
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	URL        string     `json:"url"`
	Version    string     `json:"version"`
	ZonesURL   string     `json:"zones_url"`
}

// Zone implements the common set of fields for authoritative and recursor zones.
// It needs to be inherited to work with the API, generally.
type Zone struct {
	// ID is explicitly excluded because it appears you can't set it to a value which isn't the exact same as Name.
	// PowerDNS seems happy to handle requests without it, so we don't bother calculating it either.
	//ID   string `json:"id"`
	Name string `json:"name"`
	// Type is specified in the spec but doesn't seem to appear in the JSON.
	Type string `json:"type,omitempty"`
	// URL is a "calculated" field that can be returned. It should be ignored from comparisons.
	URL string `json:"url,omitempty"`
	//Kind   string  `json:"kind"`
	RRsets RRsets `json:"rrsets"`
}

// HeaderEquals compares static zone header information only. It ignores RRsets, Type, URL
func (z *Zone) HeaderEquals(a Zone) bool {
	return z.Name == a.Name
}

// RRsets implements a collection of RRsets to allow helper methods
type RRsets []RRset

// RRsets makes a value-based copy of the containing RRsets
func (rrs RRsets) Copy() RRsets {
	result := make(RRsets, len(rrs))
	for _, rr := range rrs {
		result = append(result, rr.Copy())
	}
	return result
}

// ToMap converts an RRsets list to a map
func (rrs RRsets) ToMap() map[RRsetUniqueName]RRset {
	r := make(map[RRsetUniqueName]RRset, len(rrs))

	for _, v := range rrs {
		r[v.UniqueName()] = v.Copy()
	}

	return r
}

// Difference returns RRsets which are in this RRset but not in b down to the Record level.
// i.e. two identical RRs with different records will result in that RR being included in the
// result with only those records missing from this RRset.
func (rrs RRsets) Difference(b RRsets) RRsets {
	us := rrs.ToMap()
	them := b.ToMap()
	result := RRsets{}

	for k, v := range us {
		// If key missing entirely, add it...
		if thereV, found := them[k]; !found {
			result = append(result, v.Copy())
		} else {
			hasDifferences := false
			// Has record differences?
			recordDifferences := v.Records.Difference(thereV.Records)
			if len(recordDifferences) > 0 {
				hasDifferences = true
			}

			// Has header differences?
			if v.TTL != thereV.TTL || v.ChangeType != thereV.ChangeType {
				hasDifferences = true
			}

			// Note: Ignore name/type - should/must be the same

			// Build a "difference" RRset and add it.
			if hasDifferences {
				diffrr := v.Copy()
				diffrr.Records = recordDifferences
				result = append(result, v.Copy())
			}
		}
	}

	return result
}

// IsSubsetOf returns true if all RRsets in this collection are also in b. Differences in records even if they are
// inclusive will cause this to return false.
func (rrs RRsets) IsSubsetOf(b RRsets) bool {
	return len(rrs.Difference(b)) == 0
}

// Intersection returns RRsets which are in this RRset and b down to the Record level.
func (rrs RRsets) Intersection(b RRsets) RRsets {
	us := rrs.ToMap()
	them := b.ToMap()
	result := RRsets{}

	for k, v := range us {
		if thereV, found := them[k]; found {
			if v.TTL != thereV.TTL {
				continue
			}

			if v.ChangeType != thereV.ChangeType {
				continue
			}

			intersectingRecords := v.Records.Intersection(thereV.Records)

			intersectingRr := v.Copy()
			v.Records = intersectingRecords

			result = append(result, intersectingRr)
		}
	}

	return result
}

// Merge returns the Union of this rrset with b. Where header fields conflict, they are resolved in favor of
// this rrset.
func (rrs RRsets) Merge(b RRsets) RRsets {
	union := map[RRsetUniqueName]RRset{}

	// Copy our side.
	for _, v := range rrs {
		union[v.UniqueName()] = v
	}

	// Copy there side.
	for _, v := range b {
		un := v.UniqueName()
		if _, found := union[un]; !found {
			union[un] = v
		} else {
			// Merge there side.
			existing := union[un]
			existing.Records = union[un].Records.Union(v.Records)
			union[un] = existing
		}
	}

	result := make(RRsets, len(union))

	for _, v := range union {
		result = append(result, v)
	}
	return result
}

// RRsetUniqueName is the name and type of an RRset - sufficient to uniquely
// distinguish is.
type RRsetUniqueName struct {
	Name string
	Type string
}

// RRsetChangeType is a fixed set of string constants used when patching zones.
type RRsetChangeType string

// nolint: golint
const (
	RRsetReplace RRsetChangeType = "REPLACE"
	RRSetDelete  RRsetChangeType = "DELETE"
)

// RRset implements common RRset struct for Authoritative and Recursor APIs.
type RRset struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	TTL        int             `json:"ttl"`
	Records    Records         `json:"records"`
	ChangeType RRsetChangeType `json:"changetype,omitempty"` // Only relevant if patching
}

// Copy makes a copy of the RRset
func (rr *RRset) Copy() RRset {
	copy := *rr
	copy.Records = rr.Records.Copy()

	return copy
}

// Merge returns an RRset using the header fields of this RRset and the union'd records of b.
func (rr *RRset) Merge(b RRset) RRset {
	result := *rr // Tiny hack to avoid a double copy of records. Take note!
	result.Records = rr.Records.Union(b.Records)
	return result
}

// UniqueName returns a populated RRsetUniqueName for this RRset
func (rr *RRset) UniqueName() RRsetUniqueName {
	return RRsetUniqueName{
		rr.Name,
		rr.Type,
	}
}

// Records represents a collection of records.
type Records []Record

// ToMap returns the Records collections as a map of unique elements
func (r Records) ToMap() map[Record]struct{} {
	result := make(map[Record]struct{})
	for _, v := range r {
		result[v.Copy()] = struct{}{}
	}
	return result
}

// Difference returns the records which are in this Records collections but not in b.
func (r Records) Difference(b Records) Records {
	us := r.ToMap()
	them := b.ToMap()
	results := Records{}

	for k := range us {
		if _, found := them[k]; !found {
			results = append(results, k.Copy())
		}
	}

	return results
}

// Intersection returns the records which are in this Records collections and b.
func (r Records) Intersection(b Records) Records {
	us := r.ToMap()
	them := b.ToMap()
	results := Records{}

	for k := range us {
		if _, found := them[k]; found {
			results = append(results, k.Copy())
		}
	}

	return results
}

// Union returns Records consisting of the merged contents of both Records collections.
func (r Records) Union(b Records) Records {
	us := r.ToMap()
	them := b.ToMap()
	union := make(map[Record]struct{})

	for k := range us {
		union[k] = struct{}{}
	}

	for k := range them {
		union[k] = struct{}{}
	}

	results := Records{}

	for k := range union {
		results = append(results, k.Copy())
	}

	return results
}

// IsSubsetOf returns true if all records in this collection are also in b.
func (r Records) IsSubsetOf(b Records) bool {
	return len(r.Difference(b)) == 0
}

// Copy makes a value-based copy of Records element
func (r Records) Copy() Records {
	result := make(Records, len(r))
	for _, v := range r {
		result = append(result, v.Copy())
	}
	return result
}

// Record struct
type Record struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
	SetPtr   bool   `json:"set-ptr"`
}

// Copy makes a value-based copy of a Record
func (r *Record) Copy() Record {
	return *r
}

// Comment record which can be attached to RRsets
type Comment struct {
	Content    string    `json:"content"`
	Account    string    `json:"account"`
	ModifiedAt time.Time `json:"modified_at"`
}

// Copy makes a value based copy of a Comment
func (c *Comment) Copy() Comment {
	return *c
}
