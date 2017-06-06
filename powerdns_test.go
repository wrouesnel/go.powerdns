package powerdns

import (
	. "gopkg.in/check.v1"
)

type PowerDNSSuite struct{}

var _ = Suite(&PowerDNSSuite{})

// TestCompilation simply pulls in the objects to force a build test. It will be replaced.
func (s *PowerDNSSuite) TestCompilation(c *C) {
	NewClient("http://localhost:8080", "powerdns", true)
}
