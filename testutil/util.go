package testutil

import (
	"fmt"
	"github.com/drhodes/golorem"
	"math/rand"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
	"strings"
)

// Temporary list of dnstypes
var dnsTypes []string = []string{
	"A",
	"AAAA",
	"AFSDB",
	"APL",
	"CAA",
	"CDNSKEY",
	"CDS",
	"CERT",
	"CNAME",
	"DHCID",
	"DLV",
	"DNAME",
	"DNSKEY",
	"DS",
	"HIP",
	"IPSECKEY",
	"KEY",
	"KX",
	"LOC",
	"MX",
	"NAPTR",
	"NS",
	"NSEC",
	"NSEC3",
	"NSEC3PARAM",
	"OPENPGPKEY",
	"PTR",
	"RRSIG",
	"RP",
	"SIG",
	"SOA",
	"SRV",
	"SSHFP",
	"TA",
	"TKEY",
	"TLSA",
	"TSIG",
	"TXT",
	"URI",
}

func DnsTypes() []string {
	return dnsTypes[:]
}

// MakeRecords makes a bunch of dummy records (A style only for now)
func MakeRecords() shared.Records {
	// Make records
	records := shared.Records{}
	for i := 0; i < 1+rand.Intn(100); i++ {
		record := shared.Record{
			Disabled: rand.Intn(1) == 1,
			Content:  fmt.Sprintf("%d.%d.%d.%d", rand.Intn(254), rand.Intn(254), rand.Intn(254), rand.Intn(254)),
		}
		records = append(records, record)
	}
	return records
}

// MakeRRsets makes RRsets in the given zoneName (can be a blank string)
func MakeRRsets(zoneName string) shared.RRsets {
	rrsets := shared.RRsets{}
	for i := 0; i < 1+rand.Intn(100); i++ {
		rrset := shared.RRset{
			Name:    strings.Join([]string{lorem.Host(), zoneName}, "."),
			Type:    dnsTypes[rand.Intn(len(dnsTypes))],
			TTL:     rand.Uint32(),
			Records: MakeRecords(),
		}

		rrsets = append(rrsets, rrset)
	}
	return rrsets
}

// MakeZone makes a dummy test zone consisting of lorem-ipsum hostnames
func MakeZone() shared.Zone {
	zoneName := lorem.Host()
	z := shared.Zone{
		Name: zoneName,
		RRsets: MakeRRsets(zoneName),
	}
	return z
}