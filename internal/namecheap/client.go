// Package namecheap is a minimal client for the Namecheap XML API.
//
// Every call hits xml.response with five global parameters (ApiUser, ApiKey,
// UserName, ClientIp, Command). The caller's ClientIp must be whitelisted in the
// Namecheap dashboard and API access must be enabled on the account.
package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/connectify-studio/framework/httpx"
)

const (
	prodHost    = "https://api.namecheap.com/xml.response"
	sandboxHost = "https://api.sandbox.namecheap.com/xml.response"
)

// Credentials holds the global auth parameters required on every request.
type Credentials struct {
	APIUser  string
	APIKey   string
	UserName string
	ClientIP string
}

// Client talks to the Namecheap API.
type Client struct {
	http     *httpx.Client
	endpoint string
	creds    Credentials
}

// New builds a Client. When sandbox is true it targets the Namecheap sandbox.
func New(hc *httpx.Client, creds Credentials, sandbox bool) *Client {
	endpoint := prodHost
	if sandbox {
		endpoint = sandboxHost
	}
	return &Client{http: hc, endpoint: endpoint, creds: creds}
}

// apiResponse is the common envelope returned by every command.
type apiResponse struct {
	XMLName xml.Name `xml:"ApiResponse"`
	Status  string   `xml:"Status,attr"`
	Errors  []struct {
		Number string `xml:"Number,attr"`
		Text   string `xml:",chardata"`
	} `xml:"Errors>Error"`
	DomainGetList struct {
		Domains []Domain `xml:"Domain"`
	} `xml:"CommandResponse>DomainGetListResult"`
	Paging struct {
		TotalItems  int `xml:"TotalItems"`
		CurrentPage int `xml:"CurrentPage"`
		PageSize    int `xml:"PageSize"`
	} `xml:"CommandResponse>Paging"`
}

// Domain is a single entry from domains.getList.
type Domain struct {
	ID         string `xml:"ID,attr"`
	Name       string `xml:"Name,attr"`
	Created    string `xml:"Created,attr"`
	Expires    string `xml:"Expires,attr"`
	IsExpired  string `xml:"IsExpired,attr"`
	IsLocked   string `xml:"IsLocked,attr"`
	AutoRenew  string `xml:"AutoRenew,attr"`
	WhoisGuard string `xml:"WhoisGuard,attr"`
}

// GetListOptions are the optional filters for GetDomains.
type GetListOptions struct {
	Page       int
	PageSize   int
	SearchTerm string
}

// GetDomains calls namecheap.domains.getList and returns the domains for the
// requested page only. Use GetAllDomains to retrieve every page.
func (c *Client) GetDomains(ctx context.Context, opts GetListOptions) ([]Domain, error) {
	resp, err := c.do(ctx, c.listParams(opts))
	if err != nil {
		return nil, err
	}
	return resp.DomainGetList.Domains, nil
}

// GetAllDomains calls namecheap.domains.getList repeatedly, following the
// response paging until every domain has been retrieved. The Namecheap API caps
// PageSize at 100 and defaults to 20, so accounts with more domains than a
// single page must be paginated. opts.Page, if set, is the page to start from.
func (c *Client) GetAllDomains(ctx context.Context, opts GetListOptions) ([]Domain, error) {
	page := max(opts.Page, 1)

	var all []Domain
	for {
		o := opts
		o.Page = page
		resp, err := c.do(ctx, c.listParams(o))
		if err != nil {
			return nil, err
		}
		all = append(all, resp.DomainGetList.Domains...)

		p := resp.Paging
		// Stop when paging metadata is missing/unusable or we've covered the
		// reported total. The final guard on an empty page avoids looping
		// forever if the server returns inconsistent paging.
		if p.PageSize <= 0 || p.TotalItems <= 0 || page*p.PageSize >= p.TotalItems {
			break
		}
		if len(resp.DomainGetList.Domains) == 0 {
			break
		}
		page++
	}
	return all, nil
}

// listParams builds the query for a single namecheap.domains.getList page.
func (c *Client) listParams(opts GetListOptions) url.Values {
	q := c.baseParams("namecheap.domains.getList")
	if opts.Page > 0 {
		q.Set("Page", strconv.Itoa(opts.Page))
	}
	if opts.PageSize > 0 {
		q.Set("PageSize", strconv.Itoa(opts.PageSize))
	}
	if opts.SearchTerm != "" {
		q.Set("SearchTerm", opts.SearchTerm)
	}
	return q
}

// baseParams seeds the query with the five global parameters.
func (c *Client) baseParams(command string) url.Values {
	q := url.Values{}
	q.Set("ApiUser", c.creds.APIUser)
	q.Set("ApiKey", c.creds.APIKey)
	q.Set("UserName", c.creds.UserName)
	q.Set("ClientIp", c.creds.ClientIP)
	q.Set("Command", command)
	return q
}

func (c *Client) do(ctx context.Context, q url.Values) (*apiResponse, error) {
	u := c.endpoint + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	parsed, err := parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTP %d: %w", resp.Status, err)
	}
	return parsed, nil
}

// parse decodes a Namecheap XML envelope and turns an error status into a Go
// error. Exported behaviour is exercised via ParseDomainList in tests.
func parse(body []byte) (*apiResponse, error) {
	var parsed apiResponse
	if err := xml.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parsing Namecheap response: %w", err)
	}
	if parsed.Status != "OK" {
		if len(parsed.Errors) > 0 {
			e := parsed.Errors[0]
			return nil, fmt.Errorf("namecheap API error %s: %s", e.Number, e.Text)
		}
		return nil, fmt.Errorf("namecheap API returned status %q", parsed.Status)
	}
	// Namecheap returns dates as mm/dd/yyyy; normalize to ISO 8601 (yyyy-mm-dd)
	// so every output format (table, JSON, CSV) is consistent and sortable.
	for i := range parsed.DomainGetList.Domains {
		d := &parsed.DomainGetList.Domains[i]
		d.Created = toISODate(d.Created)
		d.Expires = toISODate(d.Expires)
	}
	return &parsed, nil
}

// toISODate converts a Namecheap mm/dd/yyyy date to yyyy-mm-dd. Values that
// don't match the expected layout are returned unchanged.
func toISODate(s string) string {
	t, err := time.Parse("01/02/2006", s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02")
}

// ParseDomainList decodes a domains.getList XML payload into domains. Provided
// for testing and reuse without performing a network call.
func ParseDomainList(body []byte) ([]Domain, error) {
	parsed, err := parse(body)
	if err != nil {
		return nil, err
	}
	return parsed.DomainGetList.Domains, nil
}
