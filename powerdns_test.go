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

	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"io"

	"github.com/wrouesnel/go.powerdns/pdnstypes/authoritative"
	"golang.org/x/tools/go/gcimporter15/testdata"
)

const (
	testAPIKey = "powerdns"
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
			"http_proxy":  &httpProxy,
			"https_proxy": &httpsProxy,
			"DOCKER_PREFIX" : &dockerPrefix,
			"API_KEY" : testAPIKey,
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
		Image: s.imageID
	}
	hostConfig := &container.HostConfig{}
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
		Follow: true,
	})

	// todo: maybe make this use c.Logf instead so it integrates with tests better
	go func(rdr io.ReadCloser) {
		io.Copy(os.Stdout, rdr)
	}(logRdr)

	c.Logf("Started container for test: %v", s.containerID)
}

// TestRawRequests initializes and tests using the data structures directly.
func (s *AuthoritativeSuite) TestRawRequests (c *C) {
	endpoint := fmt.Sprintf("http://%s:8080", s.containerIP(c))

	pdnsCli, err := NewClient(endpoint, testAPIKey, true)
	c.Assert(err, IsNil)

	// List zones (should be 0)
	listErr := pdnsCli.DoRequest("zones", "GET", nil, []authoritative.ZoneResponse{})
	c.Assert(listErr, IsNil, Commentf("Failed to list zones"))

	// Create zone

	// Create zone with contents

	// List zones

	// Add records to zone.

	// List Records

	// Remove records from zone.

	// Delete zone.


}
