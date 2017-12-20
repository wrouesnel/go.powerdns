package shared

import (
	"fmt"
	"time"
)

// Error struct
type Error struct {
	Message string  `json:"error"`
	Errors  []Error `json:"errors,omit_empty"`
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
	Type string `json:"type,omit_empty"`
	// URL is a "calculated" field that can be returned. It should be ignored from comparisons.
	URL  string `json:"url,omit_empty"`
	//Kind   string  `json:"kind"`
	RRsets RRsets `json:"rrsets"`
}

// HeaderEquals compares static zone header information only. It ignores RRsets, Type, URL
func (z *Zone) HeaderEquals(a Zone) bool {
	return z.Name == a.Name
}

// RRsets implements a collection of RRsets to allow helper methods
type RRsets []RRset

// ToNameTypeMap converts an RRsets list to a name-type nested map structure
func (rrs RRsets) ToNameTypeMap() map[string]map[string]RRset {
	r := make(map[string]map[string]RRset, len(rrs))

	for _, v := range rrs {
		typeMap, found := r[v.Name]
		if !found {
			typeMap = make(map[string]RRset)
			r[v.Name] = typeMap
		}

		if _, rrsetFound := typeMap[v.Type]; !rrsetFound {
			typeMap[v.Type] = v
		}
	}

	return r
}

// ToTypeNameMap converts an RRsets list to a type-name nested map structure
func (rrs RRsets) ToTypeNameMap() map[string]map[string]RRset {
	r := make(map[string]map[string]RRset, len(rrs))

	for _, v := range rrs {
		nameMap, found := r[v.Type]
		if !found {
			nameMap = make(map[string]RRset)
			r[v.Type] = nameMap
		}

		if _, rrsetFound := nameMap[v.Name]; !rrsetFound {
			nameMap[v.Name] = v
		}
	}

	return r
}

// RRset implements common RRset struct for Authoritative and Recursor APIs.
type RRset struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	TTL        int      `json:"ttl"`
	Records    []Record `json:"records"`
	ChangeType string   `json:"changetype,omit_empty"` // Only relevant if patching
}

// Record struct
type Record struct {
	Content  string `json:"content"`
	Disabled bool   `json:"disabled"`
	// set-ptr is explicitly ignored because it's rules are so specific at the moment
}

// Comment record which can be attached to RRsets
type Comment struct {
	Content    string    `json:"content"`
	Account    string    `json:"account"`
	ModifiedAt time.Time `json:"modified_at"`
}

// RRsets struct
//type RRsets struct {
//	Sets []RRset `json:"rrsets"`
//}
