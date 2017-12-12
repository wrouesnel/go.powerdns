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
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	URL    string  `json:"url"`
	Kind   string  `json:"kind"`
	RRsets []RRset `json:"rrsets"`
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
