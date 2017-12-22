package powerdns

import (
	. "gopkg.in/check.v1"

	"context"

	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"

	"encoding/json"
	"fmt"
	"os"
	"testing"

	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/idtools"

	"bufio"
	"net/http"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/errwrap"
	"github.com/prometheus/prometheus/util/httputil"
	"github.com/wrouesnel/go.powerdns/pdnstypes/authoritative"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"

	"math/rand"
	"strings"

	"bytes"

	lorem "github.com/drhodes/golorem"
)

const (
	testAPIKey       = "powerdns"
	containerTimeout = time.Second * 10
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

// formatWrapErr makes handling errwrap returns easier in tests
func formatWrapErr(c *C, err error) {
	if err != nil {
		errwrap.Walk(err, func(err error) { c.Logf("ERROR: %v", err) })
	}
}

func jsonFmt(c *C, inp interface{}) string {
	outp, err := json.Marshal(inp)
	c.Assert(err, IsNil)
	outbuf := bytes.NewBuffer(nil)

	jerr := json.Indent(outbuf, outp, "", "  ")
	c.Assert(jerr, IsNil)

	return outbuf.String()
}

// Add an environment variable to the build-args
func addBuildEnv(BuildArgs map[string]*string, envName string) {
	r := os.Getenv(envName)
	BuildArgs[envName] = &r
}

// checkRRsetDiff compares RRSets and returns only if they differ by records, not TTL
func rrsetDiffContainsRecords(rrsetDiff shared.RRsets) bool {
	// HACK: Sometimes PowerDNS seems to change TTLs by a few seconds in the response. We obviously don't care about
	// this so here we check if there are any record differences instead.
	for _, rr := range rrsetDiff {
		if len(rr.Records) > 0 {
			return true
		}
	}
	return false
}

// AuthoritativeSuite is a set of integration tests run against PowerDNS. A new container is initialized per-test,
// so it's structure does consist of multiple functional tests per test.
type AuthoritativeSuite struct {
	dockerCli   *client.Client
	imageID     string
	containerID string
}

var _ = Suite(&AuthoritativeSuite{})

// testContainerIP returns the IP of the currently running test container.
func (s *AuthoritativeSuite) containerIP(c *C) string {
	ctx := context.Background()
	resp, err := s.dockerCli.ContainerInspect(ctx, s.containerID)
	if err != nil {
		panic(err)
	}

	return resp.NetworkSettings.IPAddress
}

// SetUpSuite builds a PowerDNS authoritative image to use in tests.
func (s *AuthoritativeSuite) SetUpSuite(c *C) {
	c.Log("Building test docker container")
	// docker build and run the authoritative container
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	s.dockerCli = cli

	contextDir, relDockerfile, cerr := build.GetContextFromLocalDir("test/pdns_authoritative",
		"test/pdns_authoritative/Dockerfile")
	if cerr != nil {
		panic(cerr)
	}

	// read from a directory into tar archive
	excludes, rerr := build.ReadDockerignore(contextDir)
	c.Assert(rerr, IsNil)

	if verr := build.ValidateContextDirectory(contextDir, excludes); verr != nil {
		c.Fatalf("error checking context: '%s'.", verr)
	}

	// And canonicalize dockerfile name to a platform-independent one
	relDockerfile, aerr := archive.CanonicalTarNameForPath(relDockerfile)
	if aerr != nil {
		c.Fatalf("cannot canonicalize dockerfile path %s: %v", relDockerfile, aerr)
	}

	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
	buildCtx, terr := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.IDPair{UID: 0, GID: 0},
	})
	c.Assert(terr, IsNil)

	c.Log("Sending build context to docker")

	buildOptions := types.ImageBuildOptions{
		BuildArgs: map[string]*string{},
	}
	addBuildEnv(buildOptions.BuildArgs, "http_proxy")
	addBuildEnv(buildOptions.BuildArgs, "https_proxy")
	addBuildEnv(buildOptions.BuildArgs, "DOCKER_PREFIX")
	addBuildEnv(buildOptions.BuildArgs, "PDNS_AUTH_REPO_TAG")

	response, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
	c.Assert(err, IsNil)
	defer response.Body.Close()

	imageID := ""
	aux := func(auxJSON *json.RawMessage) {
		var result types.IDResponse
		if err := json.Unmarshal(*auxJSON, &result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse aux message: %s", err)
		} else {
			imageID = result.ID
		}
	}

	// Show the docker build log
	err = jsonmessage.DisplayJSONMessagesStream(response.Body, os.Stdout, os.Stdout.Fd(), false, aux)
	c.Assert(err, IsNil)

	// Set the image ID of the build
	s.imageID = imageID
	c.Logf("Build Image: %v", s.imageID)
}

// SetUpTest spawns a new pdns_authoritative server for each test in this harness.
func (s *AuthoritativeSuite) SetUpTest(c *C) {
	c.Logf("Starting pdns_authoritative docker container with image: %v", s.imageID)

	ctx := context.Background()

	containerConfig := &container.Config{
		Image: s.imageID,
		Env: []string{
			fmt.Sprintf("API_KEY=%s", testAPIKey),
		},
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true,
	}
	netConfig := &network.NetworkingConfig{}

	resp, cerr := s.dockerCli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, "")
	if cerr != nil {
		panic(cerr)
	}

	if err := s.dockerCli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	s.containerID = resp.ID

	// Stream the logs to stdout so we can see what's happening
	logRdr, err := s.dockerCli.ContainerLogs(ctx, s.containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		panic(err)
	}

	// todo: maybe make this use c.Logf instead so it integrates with tests better
	go func(rdr io.ReadCloser) {
		bio := bufio.NewReader(rdr)
		for {
			line, err := bio.ReadString('\n')
			if err != nil {
				break
			}
			c.Logf("CONTAINER: %s", line)
		}
	}(logRdr)

	c.Logf("Started container for test: %v", s.containerID)
	c.Logf("Waiting for PowerDNS to startup:")

	pingEndpoint := fmt.Sprintf("http://%s:8080/api/v1/servers/localhost", s.containerIP(c))

	client := httputil.NewClient(httputil.NewDeadlineRoundTripper(time.Second, nil))

	pingReq, err := http.NewRequest("GET", pingEndpoint, nil)
	if err != nil {
		panic(err)
	}
	pingReq.Header["Content-Type"] = []string{"application/json"}
	pingReq.Header["Accept"] = []string{"application/json"}
	pingReq.Header["X-API-Key"] = []string{testAPIKey}

	containerTimeoutCh := time.After(containerTimeout)
	tickerCh := time.Tick(time.Second)

	for {
		result := func() bool {
			resp, err := client.Do(pingReq)
			if err == nil {
				defer resp.Body.Close()
				c.Logf("Still waiting for PowerDNS to start: status %v", resp.StatusCode)
				if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
					c.Logf("PowerDNS Authoritative container is now listening.")
					return true
				}
			}
			select {
			case <-containerTimeoutCh:
				c.Errorf("PowerDNS Authoritative container did not startup within: %v", containerTimeout)
				c.FailNow()
			case <-tickerCh:
			}
			return false
		}()
		if result {
			break
		}
	}

	containerTimeoutCh = nil
	tickerCh = nil

}

func (s *AuthoritativeSuite) TearDownTest(c *C) {
	c.Logf("Stopping container for test: %v", s.containerID)

	ctx := context.Background()

	// Setup container stop waiting...
	statusCh, errCh := s.dockerCli.ContainerWait(ctx, s.containerID, container.WaitConditionNotRunning)

	// Send a container kill
	if err := s.dockerCli.ContainerKill(ctx, s.containerID, "KILL"); err != nil {
		c.Logf("Failed to stop container: %v", s.containerID)
		panic(err)
	}

	// Wait for it to stop
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
		c.Logf("Stopped: %v", s.containerID)
	}

	s.containerID = ""
}

// TestRawRequests initializes and tests using the data structures directly.
func (s *AuthoritativeSuite) TestRawRequests(c *C) {
	endpoint := fmt.Sprintf("http://%s:8080", s.containerIP(c))

	pdnsCli, err := NewClient(endpoint, testAPIKey, true)
	c.Assert(err, IsNil)

	// List zones (should be 0)
	s.testRawRequestsListZones(c, pdnsCli, 0)

	// Create zone
	s.testRawRequestsCreateZone(c, pdnsCli, "test.zone.")

	// List zones
	s.testRawRequestsListZones(c, pdnsCli, 1)

	// Create a bunch more zones
	//createdZones := []string{}
	for i := 0; i < 30; i++ {
		host := lorem.Host()
		//createdZones = append(createdZones, host)
		s.testRawRequestsCreateZone(c, pdnsCli, fmt.Sprintf("%s.", host))
	}

	// Got the expected number of zones?
	s.testRawRequestsListZones(c, pdnsCli, 31)

	// Create zone with contents
	s.testRawRequestsCreateZoneWithContents(c, pdnsCli, "populated.test.zone.")

	// Add records to zone.
	addedRRs := s.testRawRequestsAddRecordsToZone(c, pdnsCli, "test.zone.")

	// List Records and check for RRs added previously
	s.testRawRequestsListRecordsInZone(c, pdnsCli, "test.zone.", addedRRs)

	// Remove records from zone.
	s.testRawRequestsRemoveRecordsFromZone(c, pdnsCli, "test.zone.", addedRRs)

	// Delete zone.
	s.testRawRequestsDeleteZone(c, pdnsCli, "test.zone.")

}

// testRawRequestsListZonesNone tests listing zones when there none
func (s *AuthoritativeSuite) testRawRequestsListZones(c *C, pdnsCli *Client, numZones int) {
	zoneList := []authoritative.ZoneResponse{}
	listErr := pdnsCli.DoRequest("zones", "GET", nil, &zoneList)
	formatWrapErr(c, listErr)

	c.Assert(listErr, IsNil, Commentf("Failed to list zones"))
	c.Assert(len(zoneList), Equals, numZones, Commentf("Zone list was not %v length?\nGot:\n%s",
		numZones, spew.Sdump(zoneList)))
}

func (s *AuthoritativeSuite) testRawRequestsCreateZone(c *C, pdnsCli *Client, zoneName string) {
	// Generate some nameservers
	nameservers := []string{}
	for i := 1 + rand.Intn(20); i > 0; i-- {
		nameservers = append(nameservers, fmt.Sprintf("ns%v.%s", i, zoneName))
	}

	createZoneRequest := authoritative.ZoneRequestNative{
		Zone: authoritative.Zone{
			Zone: shared.Zone{
				Name: zoneName,
			},
			Kind:       authoritative.KindNative,
			SoaEdit:    authoritative.SoaEditValueInceptionIncrement,
			SoaEditAPI: authoritative.SoaEditValueInceptionIncrement,
		},
		Nameservers: nameservers,
	}
	createZoneResponse := authoritative.ZoneResponse{}
	createErr := pdnsCli.DoRequest("zones", "POST", &createZoneRequest, &createZoneResponse)
	formatWrapErr(c, createErr)

	c.Assert(createErr, IsNil, Commentf("Failed to create a new zone: Go:\n%s\nJSON:%s\n",
		spew.Sdump(createZoneRequest), jsonFmt(c, &createZoneRequest)))

	c.Assert(createZoneResponse.Zone.HeaderEquals(createZoneRequest.Zone), Equals, true,
		Commentf("returned zone not equivalent to request: Sent: %s\nGot: %s\n",
			spew.Sdump(createZoneRequest.Zone),
			spew.Sdump(createZoneResponse.Zone)))
}

func (s *AuthoritativeSuite) testRawRequestsCreateZoneWithContents(c *C, pdnsCli *Client, zoneName string) {
	// Generate some dummy records
	rrsets := shared.RRsets{}
	for i := 0; i < rand.Intn(100); i++ {
		host := strings.Join([]string{lorem.Host(), zoneName}, ".")

		records := []shared.Record{}
		for i := 0; i < 1 + rand.Intn(100); i++ {
			record := shared.Record{
				Disabled: rand.Intn(1) == 1,
				Content:  fmt.Sprintf("%d.%d.%d.%d", rand.Intn(254), rand.Intn(254), rand.Intn(254), rand.Intn(254)),
			}
			records = append(records, record)
		}

		newRR := shared.RRset{
			Name:    host,
			Type:    "A",
			TTL:     uint32(rand.Intn(2147483647)),
			Records: records,
		}
		rrsets = append(rrsets, newRR)
	}

	c.Logf("Creating zone %s with %d records", zoneName, len(rrsets))

	createZoneRequest := authoritative.ZoneRequestNative{
		Zone: authoritative.Zone{
			Zone: shared.Zone{
				Name:   zoneName,
				RRsets: rrsets,
			},
			Kind:       authoritative.KindNative,
			SoaEdit:    authoritative.SoaEditValueInceptionIncrement,
			SoaEditAPI: authoritative.SoaEditValueInceptionIncrement,
		},
		Nameservers: []string{},
	}

	createZoneResponse := authoritative.ZoneResponse{}
	createErr := pdnsCli.DoRequest("zones", "POST", &createZoneRequest, &createZoneResponse)
	formatWrapErr(c, createErr)

	c.Assert(createErr, IsNil, Commentf("Failed to create a new zone:Go:\n%s\nJSON:%s\n",
		spew.Sdump(createZoneRequest), jsonFmt(c, &createZoneRequest)))

	c.Assert(createZoneResponse.Zone.HeaderEquals(createZoneRequest.Zone), Equals, true,
		Commentf("returned zone not equivalent to request: Sent: %s\nGot: %s\n",
			spew.Sdump(createZoneRequest.Zone),
			spew.Sdump(createZoneResponse.Zone)))

	rrsetDiff := createZoneRequest.RRsets.Difference(createZoneResponse.RRsets)

	c.Assert(rrsetDiffContainsRecords(rrsetDiff), Equals, false,
		Commentf("not all RRsets from the Create Zone request were found in the response\nGo:\n%s",
			spew.Sdump(rrsetDiff)))
}

func (s *AuthoritativeSuite) testRawRequestsAddRecordsToZone(c *C, pdnsCli *Client, zoneName string) shared.RRsets {
	// Generate some dummy records
	rrsets := authoritative.PatchRRSets{}
	for i := 0; i < rand.Intn(100); i++ {
		host := strings.Join([]string{lorem.Host(), zoneName}, ".")

		records := []shared.Record{}
		for i := 0; i < 1 + rand.Intn(100); i++ {
			record := shared.Record{
				Disabled: rand.Intn(1) == 1,
				Content:  fmt.Sprintf("%d.%d.%d.%d", rand.Intn(254), rand.Intn(254), rand.Intn(254), rand.Intn(254)),
			}
			records = append(records, record)
		}

		newRR := authoritative.PatchRRSet{
			RRset: shared.RRset{
				Name:    host,
				Type:    "A",
				TTL:     uint32(rand.Intn(2147483647)),
				Records: records,
			},
			ChangeType: authoritative.RRsetReplace,
		}
		rrsets = append(rrsets, newRR)
	}

	patchZoneRequest := authoritative.PatchZoneRequest{
		RRSets: rrsets,
	}

	createErr := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "PATCH", &patchZoneRequest, nil)
	formatWrapErr(c, createErr)

	c.Assert(createErr, IsNil, Commentf("Failed add new records to zone %s: Go:\n%s\nJSON:%s\n",
		zoneName, spew.Sdump(patchZoneRequest), jsonFmt(c, &patchZoneRequest)))

	return rrsets.CopyToRRSets()
}

func (s *AuthoritativeSuite) testRawRequestsListRecordsInZone(c *C, pdnsCli *Client, zoneName string, findRRs shared.RRsets) {
	zoneResponse := &authoritative.ZoneResponse{}

	getErr := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "GET", nil, &zoneResponse)
	formatWrapErr(c, getErr)

	c.Assert(getErr, IsNil, Commentf("Could not get zone %s", zoneName))

	// Assert rrsets exist
	differences := findRRs.Difference(zoneResponse.RRsets)
	comment := `Requested RRs are not a subset of zone RRs:
Zone RRs:
%s
Search RRs:
%s
Difference JSON:
%s
`
	c.Assert(findRRs.IsSubsetOf(zoneResponse.RRsets), Equals, true,
		Commentf(comment, jsonFmt(c, &zoneResponse.RRsets), jsonFmt(c, &findRRs), jsonFmt(c, &differences)))
}

func (s *AuthoritativeSuite) testRawRequestsRemoveRecordsFromZone(c *C, pdnsCli *Client, zoneName string, removeRRs shared.RRsets) {
	// Remove records
	rrs := authoritative.NewPatchRRSets(removeRRs, authoritative.RRSetDelete)

	patchRequest := authoritative.PatchZoneRequest{RRSets: rrs}

	patchErr := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "PATCH", &patchRequest, nil)
	formatWrapErr(c, patchErr)
	c.Assert(patchErr, IsNil, Commentf("Could not delete records from zone %s", zoneName))

	// Check records were actually removed
	zoneResponse := &authoritative.ZoneResponse{}

	getErr := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "GET", nil, &zoneResponse)
	formatWrapErr(c, getErr)
	c.Assert(getErr, IsNil, Commentf("Could not get zone %s", zoneName))

	c.Assert(removeRRs.IsSubsetOf(zoneResponse.RRsets), Equals, false,
		Commentf("RRs requested for removal were still found after delete request for PATCH"))
}

//
//func (s *AuthoritativeSuite) testRawRequestsPatchRecordsInZone(c *C, pdnsCli *Client) {
//
//}
//
func (s *AuthoritativeSuite) testRawRequestsDeleteZone(c *C, pdnsCli *Client, zoneName string) {
	// Delete the zone
	delErr := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "DELETE", nil, nil)
	formatWrapErr(c, delErr)

	c.Assert(delErr, IsNil, Commentf("Failed to delete zone %s", zoneName))

	// Check that we error when trying to delete the (now non-existent) zone
	delErr2 := pdnsCli.DoRequest(fmt.Sprintf("zones/%s", zoneName), "DELETE", nil, nil)
	formatWrapErr(c, delErr2)

	c.Assert(delErr2, Not(IsNil), Commentf("Did NOT error while trying to delete previously deleted zone: %s", zoneName))
}
