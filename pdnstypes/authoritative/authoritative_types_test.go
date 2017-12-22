package authoritative_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/drhodes/golorem"
	. "github.com/wrouesnel/go.powerdns/pdnstypes/authoritative"
	"github.com/wrouesnel/go.powerdns/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type AuthTypeSuite struct{}

var _ = Suite(&AuthTypeSuite{})

func (a *AuthTypeSuite) TestZone(c *C) {
	z := Zone{
		Zone:       testutil.MakeZone(),
		Kind:       KindNative,
		SoaEdit:    SoaEditValueInceptionIncrement,
		SoaEditAPI: SoaEditValueInceptionIncrement,
		Account:    lorem.Word(5, 10),
	}

	// Self equals Self
	c.Assert(z.HeaderEquals(z), Equals, true)
	c.Assert(z.Equals(z), Equals, true)

	// Copy equals original
	zCopy := z.Copy()
	c.Assert(z.HeaderEquals(zCopy), Equals, true)
	c.Assert(z.Equals(zCopy), Equals, true)

	// Modified copy does not equal original
	z.Kind = KindMaster
	c.Assert(z.HeaderEquals(zCopy), Equals, false)
	c.Assert(z.Equals(zCopy), Equals, false)

	// Copy with modified rrsets does not equal original but headerequals does
	rrCopy := z.Copy()
	rrCopy.RRsets = testutil.MakeRRsets(rrCopy.Name)
	c.Assert(z.HeaderEquals(rrCopy), Equals, true)
	c.Assert(z.Equals(rrCopy), Equals, false)
}

func (a *AuthTypeSuite) TestPatchRRSets(c *C) {
	// Check round-tripped RRs recover the original RRs
	rrs := testutil.MakeRRsets("")
	prrs := NewPatchRRSets(rrs, RRsetReplace)
	rtrrs := prrs.CopyToRRSets()
	c.Assert(rrs.Equals(rtrrs), Equals, true)
}
