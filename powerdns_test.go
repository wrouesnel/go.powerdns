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

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/idtools"
	"io"

	"bufio"
	"github.com/hashicorp/errwrap"
	"github.com/prometheus/prometheus/util/httputil"
	"github.com/wrouesnel/go.powerdns/pdnstypes/authoritative"
	"time"
	"net/http"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
)

const (
	testAPIKey       = "powerdns"
	containerTimeout = time.Second * 10
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

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

	contextDir, relDockerfile, err := build.GetContextFromLocalDir("test/pdns_authoritative",
		"test/pdns_authoritative/Dockerfile")
	if err != nil {
		panic(err)
	}

	// read from a directory into tar archive
	excludes, err := build.ReadDockerignore(contextDir)
	c.Assert(err, IsNil)

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		c.Fatalf("error checking context: '%s'.", err)
	}

	// And canonicalize dockerfile name to a platform-independent one
	relDockerfile, err = archive.CanonicalTarNameForPath(relDockerfile)
	if err != nil {
		c.Fatalf("cannot canonicalize dockerfile path %s: %v", relDockerfile, err)
	}

	excludes = build.TrimBuildFilesFromExcludes(excludes, relDockerfile, false)
	buildCtx, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		ChownOpts:       &idtools.IDPair{UID: 0, GID: 0},
	})
	c.Assert(err, IsNil)

	c.Log("Sending build context to docker")

	httpProxy := os.Getenv("http_proxy")
	httpsProxy := os.Getenv("https_proxy")
	dockerPrefix := os.Getenv("DOCKER_PREFIX")

	buildOptions := types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"http_proxy":    &httpProxy,
			"https_proxy":   &httpsProxy,
			"DOCKER_PREFIX": &dockerPrefix,
		},
	}

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
	c.Log("Build Image: %v", s.imageID)
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

	resp, err := s.dockerCli.ContainerCreate(ctx, containerConfig, hostConfig, netConfig, "")
	if err != nil {
		panic(err)
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
			case <- tickerCh:
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

func formatWrapErr(c *C, err error) {
	if err != nil {
		errwrap.Walk(err, func(err error) { c.Logf("ERROR: %v", err) })
	}
}

// TestRawRequests initializes and tests using the data structures directly.
func (s *AuthoritativeSuite) TestRawRequests(c *C) {
	endpoint := fmt.Sprintf("http://%s:8080", s.containerIP(c))

	pdnsCli, err := NewClient(endpoint, testAPIKey, true)
	c.Assert(err, IsNil)

	// List zones (should be 0)
	zoneList := make([]authoritative.ZoneResponse,0)
	listErr := pdnsCli.DoRequest("zones", "GET", nil, &zoneList)
	formatWrapErr(c, listErr)
	c.Assert(listErr, IsNil, Commentf("Failed to list zones"))
	c.Assert(len(zoneList), Equals, 0, Commentf("Initial zone list was not 0 length?"))

	// Create zone
	createZoneRequest := authoritative.ZoneRequestNative{
		Zone: authoritative.Zone{
			Zone: shared.Zone{
				ID: "zone-test-id",
				Name: "zone.test.",
			},
			Kind: authoritative.KindNative,
			SoaEdit: authoritative.SoaEditValueInceptionIncrement,
			SoaEditAPI: authoritative.SoaEditValueInceptionIncrement,
		},
		Nameservers: []string{"ns1.zone.test.", "ns2.zone.test."},
	}
	createZoneResponse := authoritative.ZoneResponse{}

	createErr := pdnsCli.DoRequest("zones", "POST", &createZoneRequest, &createZoneResponse)
	formatWrapErr(c, createErr)
	c.Assert(createErr, IsNil, Commentf("Failed to create a new zone"))
	c.Assert(createZoneResponse.Zone, DeepEquals, createZoneRequest, Commentf("returned zone not equivalent to request"))

	// Create zone with contents

	// List zones

	// Add records to zone.

	// List Records

	// Remove records from zone.

	// Delete zone.

}
