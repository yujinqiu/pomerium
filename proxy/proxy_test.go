package proxy // import "github.com/pomerium/pomerium/proxy"

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/pomerium/pomerium/internal/config"

	"github.com/pomerium/pomerium/internal/policy"
)

var fixedDate = time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

func TestNewReverseProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		hostname, _, _ := net.SplitHostPort(r.Host)
		w.Write([]byte(hostname))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	backendHostname, backendPort, _ := net.SplitHostPort(backendURL.Host)
	backendHost := net.JoinHostPort(backendHostname, backendPort)
	proxyURL, _ := url.Parse(backendURL.Scheme + "://" + backendHost + "/")

	proxyHandler := NewReverseProxy(proxyURL)
	frontend := httptest.NewServer(proxyHandler)
	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)
	res, _ := http.DefaultClient.Do(getReq)
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	if g, e := string(bodyBytes), backendHostname; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}

func TestNewReverseProxyHandler(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		hostname, _, _ := net.SplitHostPort(r.Host)
		w.Write([]byte(hostname))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	backendHostname, backendPort, _ := net.SplitHostPort(backendURL.Host)
	backendHost := net.JoinHostPort(backendHostname, backendPort)
	proxyURL, _ := url.Parse(backendURL.Scheme + "://" + backendHost + "/")
	proxyHandler := NewReverseProxy(proxyURL)
	opts := config.NewOptions()
	opts.SigningKey = "LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSU0zbXBaSVdYQ1g5eUVneFU2czU3Q2J0YlVOREJTQ0VBdFFGNWZVV0hwY1FvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFaFBRditMQUNQVk5tQlRLMHhTVHpicEVQa1JyazFlVXQxQk9hMzJTRWZVUHpOaTRJV2VaLwpLS0lUdDJxMUlxcFYyS01TYlZEeXI5aWp2L1hoOThpeUV3PT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo="
	testPolicy := policy.Policy{From: "corp.example.com", To: "example.com", UpstreamTimeout: 1 * time.Second}
	testPolicy.Validate()

	handle, err := NewReverseProxyHandler(opts, proxyHandler, &testPolicy)
	if err != nil {
		t.Errorf("got %q", err)
	}

	frontend := httptest.NewServer(handle)

	defer frontend.Close()

	getReq, _ := http.NewRequest("GET", frontend.URL, nil)

	res, _ := http.DefaultClient.Do(getReq)
	bodyBytes, _ := ioutil.ReadAll(res.Body)
	if g, e := string(bodyBytes), backendHostname; g != e {
		t.Errorf("got body %q; expected %q", g, e)
	}
}

func testOptions() config.Options {
	authenticateService, _ := url.Parse("https://authenticate.corp.beyondperimeter.com")
	authorizeService, _ := url.Parse("https://authorize.corp.beyondperimeter.com")

	opts := config.NewOptions()
	testPolicy := policy.Policy{From: "corp.example.notatld", To: "example.notatld"}
	testPolicy.Validate()
	opts.Policies = []policy.Policy{testPolicy}
	opts.AuthenticateURL = *authenticateService
	opts.AuthorizeURL = *authorizeService
	opts.SharedKey = "80ldlrU2d7w+wVpKNfevk6fmb8otEx6CqOfshj2LwhQ="
	opts.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw="
	opts.CookieName = "pomerium"
	return opts
}

func testOptionsTestServer(uri string) config.Options {
	authenticateService, _ := url.Parse("https://authenticate.corp.beyondperimeter.com")
	authorizeService, _ := url.Parse("https://authorize.corp.beyondperimeter.com")
	// RFC 2606
	testPolicy := policy.Policy{
		From: "httpbin.corp.example",
		To:   uri,
	}
	testPolicy.Validate()
	opts := config.NewOptions()
	opts.Policies = []policy.Policy{testPolicy}
	opts.AuthenticateURL = *authenticateService
	opts.AuthorizeURL = *authorizeService
	opts.SharedKey = "80ldlrU2d7w+wVpKNfevk6fmb8otEx6CqOfshj2LwhQ="
	opts.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw="
	opts.CookieName = "pomerium"
	return opts
}

func testOptionsWithCORS(uri string) config.Options {
	testPolicy := policy.Policy{
		From:               "httpbin.corp.example",
		To:                 uri,
		CORSAllowPreflight: true,
	}
	testPolicy.Validate()
	opts := testOptionsTestServer(uri)
	opts.Policies = []policy.Policy{testPolicy}
	return opts
}

func testOptionsWithPublicAccess(uri string) config.Options {
	testPolicy := policy.Policy{
		From:                             "httpbin.corp.example",
		To:                               uri,
		AllowPublicUnauthenticatedAccess: true,
	}
	testPolicy.Validate()
	opts := testOptions()
	opts.Policies = []policy.Policy{testPolicy}
	return opts
}

func testOptionsWithPublicAccessAndWhitelist(uri string) config.Options {
	testPolicy := policy.Policy{
		From:                             "httpbin.corp.example",
		To:                               uri,
		AllowPublicUnauthenticatedAccess: true,
		AllowedEmails:                    []string{"test@gmail.com"},
	}
	testPolicy.Validate()
	opts := testOptions()
	opts.Policies = []policy.Policy{testPolicy}
	return opts
}

func TestOptions_Validate(t *testing.T) {
	good := testOptions()
	badAuthURL := testOptions()
	badAuthURL.AuthenticateURL = url.URL{}
	authurl, _ := url.Parse("http://authenticate.corp.beyondperimeter.com")
	authenticateBadScheme := testOptions()
	authenticateBadScheme.AuthenticateURL = *authurl
	authorizeBadSCheme := testOptions()
	authorizeBadSCheme.AuthorizeURL = *authurl
	authorizeNil := testOptions()
	authorizeNil.AuthorizeURL = url.URL{}
	emptyCookieSecret := testOptions()
	emptyCookieSecret.CookieSecret = ""
	invalidCookieSecret := testOptions()
	invalidCookieSecret.CookieSecret = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw^"
	shortCookieLength := testOptions()
	shortCookieLength.CookieSecret = "gN3xnvfsAwfCXxnJorGLKUG4l2wC8sS8nfLMhcStPg=="
	invalidSignKey := testOptions()
	invalidSignKey.SigningKey = "OromP1gurwGWjQPYb1nNgSxtbVB5NnLzX6z5WOKr0Yw^"
	badSharedKey := testOptions()
	badSharedKey.SharedKey = ""
	sharedKeyBadBas64 := testOptions()
	sharedKeyBadBas64.SharedKey = "%(*@389"
	missingPolicy := testOptions()
	missingPolicy.Policies = []policy.Policy{}

	tests := []struct {
		name    string
		o       config.Options
		wantErr bool
	}{
		{"good - minimum options", good, false},
		{"nil options", config.Options{}, true},
		{"authenticate service url", badAuthURL, true},
		{"authenticate service url not https", authenticateBadScheme, true},
		{"authorize service url not https", authorizeBadSCheme, true},
		{"authorize service cannot be nil", authorizeNil, true},
		{"no cookie secret", emptyCookieSecret, true},
		{"invalid cookie secret", invalidCookieSecret, true},
		{"short cookie secret", shortCookieLength, true},
		{"no shared secret", badSharedKey, true},
		{"invalid signing key", invalidSignKey, true},
		{"missing policy", missingPolicy, true},
		{"shared secret bad base64", sharedKeyBadBas64, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.o
			if err := ValidateOptions(o); (err != nil) != tt.wantErr {
				t.Errorf("Options.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNew(t *testing.T) {

	good := testOptions()
	shortCookieLength := testOptions()
	shortCookieLength.CookieSecret = "gN3xnvfsAwfCXxnJorGLKUG4l2wC8sS8nfLMhcStPg=="
	badRoutedProxy := testOptions()
	badRoutedProxy.SigningKey = "YmFkIGtleQo="
	tests := []struct {
		name      string
		opts      config.Options
		wantProxy bool
		numRoutes int
		wantErr   bool
	}{
		{"good", good, true, 1, false},
		{"empty options", config.Options{}, false, 0, true},
		{"short secret/validate sanity check", shortCookieLength, false, 0, true},
		{"invalid ec key, valid base64 though", badRoutedProxy, false, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil && tt.wantProxy == true {
				t.Errorf("New() expected valid proxy struct")
			}
			if got != nil && len(got.routeConfigs) != tt.numRoutes {
				t.Errorf("New() = num routeConfigs \n%+v, want \n%+v", got, tt.numRoutes)
			}
		})
	}
}

func Test_UpdateOptions(t *testing.T) {

	good := testOptions()
	bad := testOptions()
	bad.SigningKey = "f"
	newPolicy := policy.Policy{To: "foo.notatld", From: "bar.notatld"}
	newPolicy.Validate()
	newPolicies := []policy.Policy{
		newPolicy,
	}
	tests := []struct {
		name      string
		opts      config.Options
		newPolicy []policy.Policy
		host      string
		wantErr   bool
		wantRoute bool
	}{
		{"good", good, good.Policies, "https://corp.example.notatld", false, true},
		{"changed", good, newPolicies, "https://bar.notatld", false, true},
		{"changed and missing", good, newPolicies, "https://corp.example.notatld", false, false},
		{"bad options", bad, good.Policies, "https://corp.example.notatld", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.opts
			p, _ := New(o)

			o.Policies = tt.newPolicy
			err := p.UpdateOptions(o)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateOptions: err = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			// This is only safe if we actually can load policies
			if err == nil {
				req := httptest.NewRequest("GET", tt.host, nil)
				_, ok := p.router(req)
				if ok != tt.wantRoute {
					t.Errorf("Failed to find route handler")
					return
				}
			}
		})
	}
}
