package namecheap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/connectify-studio/framework/httpx"
)

const okPayload = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="127" Name="example.com" User="alice" Created="01/15/2020" Expires="01/15/2027" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="ENABLED" />
      <Domain ID="128" Name="foo.dev" User="alice" Created="03/03/2021" Expires="03/03/2026" IsExpired="false" IsLocked="true" AutoRenew="false" WhoisGuard="NOTPRESENT" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>2</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>20</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>`

const errPayload = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors>
    <Error Number="1011102">API Key is invalid or API access has not been enabled</Error>
  </Errors>
  <CommandResponse />
</ApiResponse>`

func TestParseDomainList(t *testing.T) {
	domains, err := ParseDomainList([]byte(okPayload))
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 2 {
		t.Fatalf("got %d domains, want 2", len(domains))
	}
	d := domains[0]
	if d.Name != "example.com" || d.AutoRenew != "true" || d.WhoisGuard != "ENABLED" {
		t.Errorf("unexpected first domain: %+v", d)
	}
	// Dates are normalized from Namecheap's mm/dd/yyyy to ISO yyyy-mm-dd.
	if d.Created != "2020-01-15" || d.Expires != "2027-01-15" {
		t.Errorf("dates not ISO-normalized: Created=%q Expires=%q", d.Created, d.Expires)
	}
	if domains[1].IsLocked != "true" {
		t.Errorf("second domain IsLocked = %q, want true", domains[1].IsLocked)
	}
}

func TestParseAPIError(t *testing.T) {
	_, err := ParseDomainList([]byte(errPayload))
	if err == nil {
		t.Fatal("expected error for ERROR status payload")
	}
	if got := err.Error(); !contains(got, "1011102") || !contains(got, "API access") {
		t.Errorf("error should surface Namecheap message, got %q", got)
	}
}

func TestToISODate(t *testing.T) {
	cases := map[string]string{
		"01/15/2020": "2020-01-15",
		"12/31/1999": "1999-12-31",
		"03/03/2021": "2021-03-03",
		"":           "",           // unchanged
		"not-a-date": "not-a-date", // unchanged
		"2020-01-15": "2020-01-15", // already ISO, left as-is
	}
	for in, want := range cases {
		if got := toISODate(in); got != want {
			t.Errorf("toISODate(%q) = %q, want %q", in, got, want)
		}
	}
}

// pagedPayload renders a getList page with the given paging metadata.
func pagedPayload(domains string, total, page, size int) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>%s</DomainGetListResult>
    <Paging>
      <TotalItems>%d</TotalItems>
      <CurrentPage>%d</CurrentPage>
      <PageSize>%d</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>`, domains, total, page, size)
}

func TestGetAllDomainsPaginates(t *testing.T) {
	const total, size = 3, 2
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("Page")
		requested = append(requested, page)
		w.Header().Set("Content-Type", "application/xml")
		switch page {
		case "1":
			fmt.Fprint(w, pagedPayload(
				`<Domain ID="1" Name="a.com" Created="01/15/2020" Expires="01/15/2027" />`+
					`<Domain ID="2" Name="b.com" Created="02/15/2020" Expires="02/15/2027" />`,
				total, 1, size))
		case "2":
			fmt.Fprint(w, pagedPayload(
				`<Domain ID="3" Name="c.com" Created="03/15/2020" Expires="03/15/2027" />`,
				total, 2, size))
		default:
			t.Errorf("unexpected page request %q", page)
		}
	}))
	defer srv.Close()

	c := &Client{http: httpx.New(), endpoint: srv.URL, creds: Credentials{}}
	domains, err := c.GetAllDomains(context.Background(), GetListOptions{PageSize: size})
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != total {
		t.Fatalf("got %d domains across pages, want %d", len(domains), total)
	}
	if len(requested) != 2 || requested[0] != "1" || requested[1] != "2" {
		t.Errorf("expected requests for pages [1 2], got %v", requested)
	}
	// Dates from every page are ISO-normalized.
	if domains[2].Created != "2020-03-15" {
		t.Errorf("last domain Created = %q, want 2020-03-15", domains[2].Created)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
