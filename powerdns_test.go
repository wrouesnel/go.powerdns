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
	"io"
	"os"

	"github.com/docker/docker/pkg/idtools"
)

type AuthoritativeSuite struct {
	imageID     string
	containerID string
	containerIP string
}

var _ = Suite(&AuthoritativeSuite{})

func (s *AuthoritativeSuite) SetUpTest(c *C) {
	// docker build and run the authoritative container
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	contextDir, relDockerfile, err := build.GetContextFromLocalDir("test/pdns_authoritative", "Dockerfile")
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

	buildOptions := types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"http_proxy":  &httpProxy,
			"https_proxy": &httpsProxy,
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
	var buildBuff io.Writer
	err = jsonmessage.DisplayJSONMessagesStream(response.Body, buildBuff, os.Stdout.Fd(), true, aux)
	c.Assert(err, IsNil)

	// Set the image ID of the build
	s.imageID = imageID
}

func (s *AuthoritativeSuite) TestCompilation(c *C) {
	NewClient("http://localhost:8080", "powerdns", true)
}
