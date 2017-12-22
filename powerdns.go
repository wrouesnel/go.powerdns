package powerdns

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"

	"fmt"
	"net"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
)

// nolint: golint
var (
	// ErrClientNilError
	ErrClientNilError            = errors.New("No URL supplied for API client.")
	ErrClientSubPathError        = errors.New("Subpath URI was badly formed.")
	ErrClientRequestParsingError = errors.New("Error parsing request parameters locally")
	ErrClientRequestIsAbs        = errors.New("Absolute URI is not allowed")
	ErrClientRequestFailed       = errors.New("Error sending request to server")
	ErrClientServerUnknownStatus = errors.New("Server returned a StatusCode it shouldn't have.")
	ErrClientServerResponse      = errors.New("Server returned an error response")
)

// ErrClientServerResponseUnreadable is returned when the server sends us something non-sensical, and includes
// the body of the response.
type ErrClientServerResponseUnreadable struct {
	serverResponse []byte
}

func (err ErrClientServerResponseUnreadable) Error() string {
	return "Server returned a response that could not be deserialized"
}

// ResponseBodyString returns the response body as a string.
func (err ErrClientServerResponseUnreadable) ResponseBodyString() string {
	if err.serverResponse != nil {
		return string(err.serverResponse)
	}
	return ""
}

// ResponseBody returns a copy of the response body, if any, which was sent by the server
func (err ErrClientServerResponseUnreadable) ResponseBody() []byte {
	r := make([]byte, len(err.serverResponse))
	copy(r, err.serverResponse)
	return r
}

const (
	apiPathString = "api/v1/"
)

// resolveAPIPath modifies the given url to have an API path from the global constant.
var resolveAPIPath func(u *url.URL) *url.URL

func init() {
	apiSubPath, err := url.Parse(apiPathString)
	if err != nil {
		panic(err)
	}
	resolveAPIPath = func(u *url.URL) *url.URL {
		return u.ResolveReference(apiSubPath)
	}
}

// Client client struct
type Client struct {
	endpoint   *url.URL
	serverPath *url.URL // Server endpoint is added to match the multi-server functionality of pdns.
	headers    http.Header
	cli        *http.Client
}

// deadlineRoundTripper utility function lifted from prometheus.httputil with a few modifications
func deadlineRoundTripper(timeout time.Duration, proxyURL *url.URL, tlsInsecure bool) http.RoundTripper {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsInsecure}, // nolint: gas
		// Set proxy (if null, then becomes a direct connection)
		Proxy: http.ProxyURL(proxyURL),
		// We need to disable keepalive, because we set a deadline on the
		// underlying connection.
		DisableKeepAlives: true,
		Dial: func(netw, addr string) (c net.Conn, err error) {
			start := time.Now()

			c, err = net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}

			if err = c.SetDeadline(start.Add(timeout)); err != nil {
				c.Close()
				return nil, err
			}

			return c, nil
		},
	}
}

// NewClient initializes an API client with some common defaults.
func NewClient(endpoint string, apiKey string, tlsInsecure bool, timeout time.Duration) (*Client, error) {
	// TLS conf
	tr := deadlineRoundTripper(timeout, nil, tlsInsecure)
	client := &http.Client{Transport: tr}

	// Decode the url
	decodedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// Set API key
	headers := http.Header{}
	headers["X-API-Key"] = []string{apiKey}

	return New(decodedURL, "localhost", client, headers)
}

// New returns a New PowerDNS API client. If cli is set to nil, the default httpClient
// is used (this will probably not work as you need to set an API key header - its also advisable to
// configure connection time outs).
func New(endpoint *url.URL, server string, cli *http.Client, headers http.Header) (*Client, error) {
	if endpoint == nil {
		return nil, ErrClientNilError
	}

	if cli == nil {
		cli = http.DefaultClient
	}

	serverPath, err := url.Parse(fmt.Sprintf("servers/%s/", server))
	if err != nil {
		return nil, errwrap.Wrap(ErrClientSubPathError, err)
	}

	if serverPath.IsAbs() {
		return nil, ErrClientRequestIsAbs
	}

	apiClient := &Client{
		endpoint:   endpoint,
		serverPath: serverPath,
		headers:    headers,
		cli:        cli,
	}

	return apiClient, nil
}

// resolveServerPath adds the configured server path component to the endpoing URL
func (p *Client) resolveServerPath(u *url.URL) *url.URL {
	return u.ResolveReference(p.serverPath)
}

// resolveRequestPath wraps all the logic needed to resolve the full URI to send a given request to a server
func (p *Client) resolveRequestPath(u *url.URL) *url.URL {
	return p.resolveServerPath(resolveAPIPath(p.endpoint)).ResolveReference(u)
}

// DoRequest executes a generic request against a sub-path of the PowerDNS API.
func (p *Client) DoRequest(subPathStr string,
	method string,
	requestType interface{},
	responseType interface{}) error {

	subPath, err := url.Parse(subPathStr)
	if err != nil {
		return errwrap.Wrap(ErrClientSubPathError, err)
	}

	if subPath.IsAbs() {
		return ErrClientRequestIsAbs
	}

	// TODO: consider making resolveServerPath implicitly handle API path resolution
	requestPath := p.resolveRequestPath(subPath)

	requestBody, jerr := json.Marshal(requestType)
	if jerr != nil {
		return errwrap.Wrap(ErrClientRequestParsingError, jerr)
	}

	httpReq, rerr := http.NewRequest(method, requestPath.String(), bytes.NewBuffer(requestBody))
	if rerr != nil {
		return errwrap.Wrap(ErrClientRequestParsingError, rerr)
	}

	// Add the headers.
	for key, values := range p.headers {
		inputHeaders := values[:]
		httpReq.Header[key] = inputHeaders
	}

	// Forcibly set the JSON content type header and Accept header since the API requires it.
	httpReq.Header["Content-Type"] = []string{"application/json"}
	httpReq.Header["Accept"] = []string{"application/json"}

	// Execute the request.
	resp, derr := p.cli.Do(httpReq)
	if derr != nil {
		return errwrap.Wrap(ErrClientRequestFailed, derr)
	}

	// Deserialize the response.
	defer resp.Body.Close() //nolint: errcheck

	respBody, ierr := ioutil.ReadAll(resp.Body)
	if ierr != nil {
		return errwrap.Wrap(ErrClientServerResponseUnreadable{respBody}, ierr)
	}

	// Check if an HTTP error code was returned, in which case we need to return an error type.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Did not get 200, so we failed. Did we get a reported fail from the server?
		if 400 <= resp.StatusCode && resp.StatusCode <= 599 {
			// Should be able to unmarshal an error type.
			responseErr := shared.Error{}
			var wrappedErr error
			if uerr := json.Unmarshal(respBody, &responseErr); uerr != nil {
				wrappedErr = errwrap.Wrap(ErrClientServerResponseUnreadable{respBody}, uerr)
			} else {
				wrappedErr = responseErr
			}
			wrappedErr = errwrap.Wrap(ErrClientServerResponse, responseErr)
			return wrappedErr
		}
		// Did not succeed, but did not recognize the status code either.
		return ErrClientServerUnknownStatus
	}

	// Success! Unmarshal into the user type (if usertype supplied)
	if responseType != nil {
		if juerr := json.Unmarshal(respBody, responseType); juerr != nil {
			return errwrap.Wrap(ErrClientServerResponseUnreadable{respBody}, juerr)
		}
	}

	return nil
}
