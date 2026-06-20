package namecheap

import "testing"

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
	if d.Name != "example.com" || d.AutoRenew != "true" || d.WhoisGuard != "ENABLED" || d.Expires != "01/15/2027" {
		t.Errorf("unexpected first domain: %+v", d)
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

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
