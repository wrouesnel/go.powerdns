package powerdns

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/hashicorp/errwrap"
	"github.com/wrouesnel/go.powerdns/pdnstypes/shared"
	"io/ioutil"
	"net/http"
	"net/url"
)

var (
	ErrClientNilError                 = errors.New("No URL supplied for API client.")
	ErrClientRequestParsingError      = errors.New("Error parsing request parameters locally")
	ErrClientRequestIsAbs             = errors.New("Absolute URI is not allowed")
	ErrClientRequestFailed            = errors.New("Error sending request to server")
	ErrClientServerUnknownStatus      = errors.New("Server returned a StatusCode it shouldn't have.")
	ErrClientServerResponseUnreadable = errors.New("Server returned a response that could not be deserialized")
	ErrClientServerResponse           = errors.New("Server returned an error response")
)

// PowerDNSClient client struct
type PowerDNSClient struct {
	endpoint *url.URL
	headers  http.Header
	cli      *http.Client
}

// NewClient initializes an API client with some common default.
func NewClient(endpoint string, apiKey string, tlsInsecure bool) (*PowerDNSClient, error) {
	// TLS conf
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsInsecure},
	}
	client := &http.Client{Transport: tr}

	// Decode the url
	decodedUrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	// Set API key
	headers := http.Header{}
	headers["X-API-Key"] = []string{apiKey}

	return New(decodedUrl, client, headers)
}

// New returns a New PowerDNS API client. If cli is set to nil, the default httpClient
// is used (this will probably not work as you need to set an API key header).
func New(endpoint *url.URL, cli *http.Client, headers http.Header) (*PowerDNSClient, error) {
	if endpoint == nil {
		return nil, ErrClientNilError
	}

	if cli == nil {
		cli = http.DefaultClient
	}

	apiClient := &PowerDNSClient{
		endpoint: endpoint,
		headers:  headers,
		cli:      cli,
	}

	return apiClient, nil
}

// DoRequest executes a generic request against a sub-path of the PowerDNS API.
func (p *PowerDNSClient) DoRequest(subPath *url.URL, method string, requestType interface{}, responseType interface{}) error {
	if subPath.IsAbs() {
		return ErrClientRequestIsAbs
	}

	requestPath := p.endpoint.ResolveReference(subPath)

	requestBody, err := json.Marshal(requestType)
	if err != nil {
		return errwrap.Wrap(ErrClientRequestParsingError, err)
	}

	httpReq, err := http.NewRequest(method, requestPath.RequestURI(), bytes.NewBuffer(requestBody))
	if err != nil {
		return errwrap.Wrap(ErrClientRequestParsingError, err)
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
	resp, err := p.cli.Do(httpReq)
	if err != nil {
		return errwrap.Wrap(ErrClientRequestFailed, err)
	}

	// Deserialize the response.
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errwrap.Wrap(ErrClientServerResponseUnreadable, err)
	}

	// Check if an HTTP error code was returned, in which case we need to return an error type.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Did not get 200, so we failed. Did we get a reported fail from the server?
		if 400 <= resp.StatusCode && resp.StatusCode <= 599 {
			// Should be able to unmarshal an error type.
			responseErr := shared.Error{}
			if err := json.Unmarshal(respBody, &responseErr); err != nil {
				return errwrap.Wrap(ErrClientServerResponseUnreadable, err)
			}
			return errwrap.Wrap(ErrClientServerResponse, err)
		}
		// Did not succeed, but did not recognize the status code either.
		return ErrClientServerUnknownStatus
	}

	// Success! Unmarshal into the user type
	if err := json.Unmarshal(respBody, responseType); err != nil {
		return errwrap.Wrap(ErrClientServerResponseUnreadable, err)
	}

	return nil
}

//func (p *PowerDNSClient) baseRequest() *http.Request {
//	http.NewRequest("", p.endpoint.String(), )
//}
//
//func (p *PowerDNSClient) GetServers() {
//	p.endpoint.ResolveReference()
//}

// AddRecord ...
//func (p *PowerDNSClient) AddRecord(name string, recordType string, ttl int, content []string) error {
//
//	return p.ChangeRecord(name, recordType, ttl, content, "REPLACE")
//}

// DeleteRecord ...
//func (p *PowerDNSClient) DeleteRecord(name string, recordType string, ttl int, content []string) error {
//
//	return p.ChangeRecord(name, recordType, ttl, content, "DELETE")
//}

// ChangeRecord ...
//func (p *PowerDNSClient) ChangeRecord(name string, recordType string, ttl int, contents []string, action string) error {
//
//	// Add trailing dot for V1 and removes it for V0
//	if p.apiVersion == 1 {
//		name = addTrailingCharacter(name, '.')
//	} else {
//		name = strings.TrimRight(name, ".")
//	}
//
//	rrset := RRset{
//		Name: name,
//		Type: recordType,
//		TTL:  ttl,
//	}
//
//	for _, content := range contents {
//		if rrset.Type == "TXT" {
//			content = "\"" + strings.Replace(content, "\"", "", -1) + "\""
//		}
//		rrset.Records = append(rrset.Records, Record{
//			Content: content,
//			Name:    name,
//			TTL:     ttl,
//			Type:    recordType,
//		})
//	}
//
//	return p.patchRRset(rrset, action)
//}

// GetRecords ...
//func (p *PowerDNSClient) GetRecords() ([]Record, error) {
//
//	var records []Record
//
//	zone := new(Zone)
//	rerr := new(Error)
//
//	resp, err := p.getSling().Path(p.path+"/servers/"+p.server+"/zones/"+p.domain).Set("X-API-Key", p.apikey).Receive(zone, rerr)
//
//	if err != nil {
//		return records, fmt.Errorf("PowerDNSClient API call has failed: %v", err)
//	}
//
//	if resp.StatusCode >= 400 {
//		rerr.Message = strings.Join([]string{resp.Status, rerr.Message}, " ")
//		return records, rerr
//	}
//
//	if len(zone.Records) > 0 {
//		for i, record := range zone.Records {
//			if record.Type == "TXT" {
//				zone.Records[i].Content = strings.Replace(record.Content, "\"", "", -1)
//			}
//		}
//		records = zone.Records
//	} else {
//		for _, rrset := range zone.RRsets {
//			for _, rec := range rrset.Records {
//				if rrset.Type == "TXT" {
//					rec.Content = strings.Replace(rec.Content, "\"", "", -1)
//				}
//				if p.apiVersion == 1 {
//					rrset.Name = strings.TrimRight(rrset.Name, ".")
//				}
//				record := Record{
//					Name:     rrset.Name,
//					Type:     rrset.Type,
//					Content:  rec.Content,
//					TTL:      rrset.TTL,
//					Disabled: rec.Disabled,
//				}
//				records = append(records, record)
//			}
//		}
//	}
//
//	return records, err
//}
//
//func (p *PowerDNSClient) patchRRset(rrset RRset, action string) error {
//
//	rrset.ChangeType = "REPLACE"
//
//	if action == "DELETE" {
//		rrset.ChangeType = "DELETE"
//	}
//
//	sets := RRsets{}
//	sets.Sets = append(sets.Sets, rrset)
//
//	rerr := new(Error)
//	zone := new(Zone)
//
//	resp, err := p.getSling().Path(p.path+"/servers/"+p.server+"/zones/").Patch(p.domain).BodyJSON(sets).Receive(zone, rerr)
//
//	if err == nil && resp.StatusCode >= 400 {
//		rerr.Message = strings.Join([]string{resp.Status, rerr.Message}, " ")
//		return rerr
//	}
//
//	if resp.StatusCode == 204 {
//		return nil
//	}
//
//	return err
//}

//func (p *PowerDNSClient) detectAPIVersion() (int, error) {
//
//	versions := new([]APIVersion)
//	info := new(ServerInfo)
//	rerr := new(Error)
//
//	resp, err := p.getSling().Path("api").Receive(versions, rerr)
//	if resp == nil && err != nil {
//		return -1, err
//	}
//
//	if resp.StatusCode == 404 {
//		resp, err = p.getSling().Path("servers/").Path(p.server).Receive(info, rerr)
//		if resp == nil && err != nil {
//			return -1, err
//		}
//	}
//
//	if resp.StatusCode != 200 {
//		rerr.Message = strings.Join([]string{resp.Status, rerr.Message}, " ")
//		return -1, rerr
//	}
//
//	if err != nil {
//		return -1, err
//	}
//
//	latestVersion := APIVersion{"", 0}
//	for _, v := range *versions {
//		if v.Version > latestVersion.Version {
//			latestVersion = v
//		}
//	}
//	p.path = p.path + latestVersion.URL
//
//	return latestVersion.Version, err
//}

//func (p *PowerDNSClient) getSling() *sling.Sling {
//
//	u := new(url.URL)
//	u.Host = p.hostname + ":" + p.port
//	u.Scheme = p.scheme
//	u.Path = p.path
//
//	// Add trailing slash if necessary
//	u.Path = addTrailingCharacter(u.Path, '/')
//
//	return sling.New().Base(u.String()).Set("X-API-Key", p.apikey)
//}
//
//func addTrailingCharacter(name string, character byte) string {
//
//	// Add trailing dot if necessary
//	if len(name) > 0 && name[len(name)-1] != character {
//		name += string(character)
//	}
//
//	return name
//}
