package recursor_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"math/rand"

	. "github.com/wrouesnel/go.powerdns/pdnstypes/recursor"
	"github.com/wrouesnel/go.powerdns/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type RecTypeSuite struct{}

var _ = Suite(&RecTypeSuite{})

func (r *RecTypeSuite) TestZone(c *C) {
	z := Zone{
		Zone:             testutil.MakeZone(),
		Servers:          testutil.MakeRandIPList(10),
		RecursionDesired: rand.Intn(1) == 1,
	}

	// self equals self
	c.Assert(z.HeaderEquals(z), Equals, true)
	c.Assert(z.Equals(z), Equals, true)

	zCopy := z.Copy()
	// copy equals self
	c.Assert(z.HeaderEquals(zCopy), Equals, true)
	c.Assert(z.Equals(zCopy), Equals, true)

	// modified copy does not equal original
	zCopy.Name = "something else"
	c.Assert(z.HeaderEquals(zCopy), Equals, false)
	c.Assert(z.Equals(zCopy), Equals, false)
}
