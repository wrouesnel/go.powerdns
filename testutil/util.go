package testutil

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/drhodes/golorem"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
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

// DnsTypes returns a list of DnsTypes as a string slice copy
func DnsTypes() []string {
	return dnsTypes[:]
}

// MakeMixedHostIPList makes a list of random hostnames and IPs as a string slice
func MakeMixedHostIPList(n int, suffix string) []string {
	src := MakeRandHostList(n, suffix)
	src = append(src, MakeRandIPList(n)...)

	for i := range src {
		j := rand.Intn(i + 1)
		src[i], src[j] = src[j], src[i]
	}
	return src
}

// MakeRandIPList makes a list of random hostnames from lorem ipsum as a string slice
func MakeRandHostList(n int, suffix string) []string {
	r := []string{}
	for i := 0; i < rand.Intn(n); i++ {
		if suffix != "" {
			r = append(r, strings.Join([]string{lorem.Host(), suffix}, "."))
		} else {
			r = append(r, lorem.Host())
		}
	}
	return r
}

// MakeRandIPList makes a list of random IP addresses as a string slice
func MakeRandIPList(n int) []string {
	r := []string{}
	for i := 0; i < rand.Intn(n); i++ {
		r = append(r, MakeRandIP())
	}
	return r
}

// MakeRandIP Makes a random IPv4 address as a string
func MakeRandIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(254), rand.Intn(254), rand.Intn(254), rand.Intn(254))
}

// MakeRecords makes a bunch of dummy records (A style only for now)
func MakeRecords() shared.Records {
	// Make records
	records := shared.Records{}
	for i := 0; i < 1+rand.Intn(100); i++ {
		record := shared.Record{
			Disabled: rand.Intn(1) == 1,
			Content:  MakeRandIP(),
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
		Name:   zoneName,
		RRsets: MakeRRsets(zoneName),
	}
	return z
}
