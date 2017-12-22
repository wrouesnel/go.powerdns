package shared

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/drhodes/golorem"
	"github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
	"github.com/wrouesnel/go.powerdns/testutil"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type SharedTypeSuite struct{}

var _ = Suite(&SharedTypeSuite{})



func (s *SharedTypeSuite) TestComment(c *C) {
	// Initialize a new comment
	comment := Comment{
		"Content",
		"Account",
		time.Now(),
	}
	// Copy the comment
	copiedComment := comment.Copy()
	// Check comments are identical
	c.Check(copiedComment, DeepEquals, comment)
	// Modify original and check they are now not equal
	comment.Content = "Different Content"
	c.Check(copiedComment, Not(DeepEquals), comment,
		Commentf("Comments still equal after modification:\n%s\n%s\n", spew.Sdump(copiedComment),
			spew.Sdump(comment)))
}

func (s *SharedTypeSuite) TestRecord(c *C) {
	record := Record{
		"Content",
		false,
		false,
	}
	// Copy the record
	copiedRecord := record.Copy()
	// Check the records are identical
	c.Check(copiedRecord, DeepEquals, record)
	record.Content = "Different Content"
	c.Check(copiedRecord, Not(DeepEquals), record,
		Commentf("Record still equal after modification:\n%s\n%s\n", spew.Sdump(copiedRecord),
			spew.Sdump(record)))
}

func (s *SharedTypeSuite) TestRecords(c *C) {
	records := testutil.MakeRecords()

	// Test ToMap
	mappedRecords := records.ToMap()
	for _, v := range records {
		_, found := mappedRecords[v]
		c.Assert(found, Equals, true, Commentf("could not find list record in map"))
	}

	// Test Equals
	records.Equals(records)

	// Make a copy
	recordsCopy := records.Copy()
	// Test the copy
	c.Assert(recordsCopy, DeepEquals, records)
	// Test equals agrees with the copy
	c.Assert(records.Equals(recordsCopy), Equals, true)

	// Edit the records copy
	for idx := range recordsCopy {
		recordsCopy[idx].Content = fmt.Sprintf("edited record %v", idx)
	}
	c.Assert(recordsCopy, Not(DeepEquals), records, Commentf("After modification, recordsCopy still equals Records"))
	// Test equals no longer agrees with the copy
	c.Assert(records.Equals(recordsCopy), Equals, false)

	// Test Difference
	diffRecordCopy := records.Copy()
	diffRecordCopy = append(diffRecordCopy,
		Record{"difference extra 1", false, false},
		Record{"difference extra 2", false, false})

	diffedRecords := diffRecordCopy.Difference(records)
	c.Assert(len(diffedRecords), Equals, 2)
	for _, v := range diffedRecords {
		c.Assert(v.Content, Matches, "difference extra \\d")
	}

	// Test IsSubsetOf against the appended copy above
	c.Assert(records.IsSubsetOf(diffRecordCopy), Equals, true)
	c.Assert(diffRecordCopy.IsSubsetOf(records), Equals, false)

	// Test Union
	c.Assert(records.Union(diffRecordCopy).Equals(diffRecordCopy), Equals, true)

	// Test intersection
	c.Assert(diffRecordCopy.Intersection(records).Equals(records), Equals, true)
}

func (s *SharedTypeSuite) TestRRSet(c *C) {
	rrset := RRset{
		Name:    "testrr.com",
		Type:    "A",
		TTL:     100000,
		Records: testutil.MakeRecords(),
	}
	// Check we equal ourselves
	c.Assert(rrset.Equals(rrset), Equals, true)

	// Check we can copy
	copiedRRset := rrset.Copy()
	c.Assert(copiedRRset.Equals(rrset), Equals, true)
	c.Assert(copiedRRset, DeepEquals, rrset)

	// Modify the copy
	copiedRRset.TTL = 1
	c.Assert(copiedRRset.Equals(rrset), Equals, false)

	// Modify it back
	copiedRRset.TTL = rrset.TTL
	c.Assert(copiedRRset.Equals(rrset), Equals, true)

	// Add some new records to the copy and check it reports different
	copiedRRset.Records = copiedRRset.Records.Union(testutil.MakeRecords())
	c.Assert(copiedRRset.Equals(rrset), Equals, false)

	// Test merging
	mergeTestRRset := rrset.Copy()
	mergeTestRRset.Name = "somethingelse.com"
	mergeTestRRset.TTL = 1
	mergeTestResult := mergeTestRRset.Merge(copiedRRset)
	// Check we kept headers
	c.Assert(mergeTestResult.UniqueName(), Equals, mergeTestRRset.UniqueName())
	// But got a union of rrsets in records
	c.Assert(mergeTestResult.Records.Equals(mergeTestRRset.Records.Union(copiedRRset.Records)), Equals, true)

	// Check unique names seem to be unique
	testMap := make(map[RRsetUniqueName]struct{})
	for i := 0; i < 30; i++ {
		rrset := RRset{
			Name:    fmt.Sprintf("domain-%s.com", uuid.NewV4().String()),
			Type:    "",
			TTL:     rand.Uint32(),
			Records: testutil.MakeRecords(),
		}

		for _, v := range testutil.DnsTypes() {
			newrr := rrset.Copy()
			rrset.Type = v
			// We should never find a match since we're generating unique names and types
			_, found := testMap[newrr.UniqueName()]
			c.Assert(found, Equals, false)
			testMap[newrr.UniqueName()] = struct{}{}
		}
	}
}

func (s *SharedTypeSuite) TestRRSets(c *C) {
	rrs := testutil.MakeRRsets(".")

	// Test ToMap
	mappedRecords := rrs.ToMap()
	for _, v := range rrs {
		_, found := mappedRecords[v.UniqueName()]
		c.Assert(found, Equals, true, Commentf("could not find list RRSet in map"))
	}

	// Test Equals
	rrs.Equals(rrs)

	// Make a copy
	rrsCopy := rrs.Copy()
	// Test the copy
	c.Assert(rrsCopy, DeepEquals, rrs)
	// Test equals agrees with the copy
	c.Assert(rrs.Equals(rrsCopy), Equals, true)

	// Edit the records copy
	for idx := range rrsCopy {
		rrsCopy[idx].Name = fmt.Sprintf("edited record %v", idx)
	}
	c.Assert(rrsCopy, Not(DeepEquals), rrs, Commentf("After modification, rrscopy still equals rrs"))
	// Test equals no longer agrees with the copy
	c.Assert(rrs.Equals(rrsCopy), Equals, false)

	// Test Difference
	diffRRSCopy := rrs.Copy()

	appendedRRs := testutil.MakeRRsets(".")
	diffRRSCopy = append(diffRRSCopy, appendedRRs...)
	diffedRecords := diffRRSCopy.Difference(rrs)
	c.Assert(len(diffedRecords), Equals, len(appendedRRs))

	appendedRRsMap := appendedRRs.ToMap()
	for _, v := range diffedRecords {
		_, found := appendedRRsMap[v.UniqueName()]
		c.Assert(found, Equals, true,
			Commentf("An appended RRset was not found in the difference between the appended set and original"))
	}

	// Test IsSubsetOf against the appended copy above
	c.Assert(rrs.IsSubsetOf(diffRRSCopy), Equals, true)
	c.Assert(diffRRSCopy.IsSubsetOf(rrs), Equals, false)

	// Test Merge
	c.Assert(rrs.Merge(diffRRSCopy).Equals(diffRRSCopy), Equals, true)
}

func (s *SharedTypeSuite) TestZone(c *C) {
	z := testutil.MakeZone()

	c.Assert(z.HeaderEquals(z), Equals, true)
	c.Assert(z.Equals(z), Equals, true)

	// Make a copy and check it matches
	b := z.Copy()
	c.Assert(z.HeaderEquals(b), Equals, true)
	c.Assert(z.Equals(b), Equals, true)

	// Edit the RRsets and check it does not match
	b.RRsets = testutil.MakeRRsets(z.Name)
	c.Assert(z.HeaderEquals(b), Equals, true)
	c.Assert(z.Equals(b), Equals, false)
}
