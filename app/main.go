package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/openrdap/api/caching"
	"github.com/openrdap/rdap/bootstrap"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

func main() {
	http.HandleFunc("/", handle)
	appengine.Main()
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not supported", 501)
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")

	s := http.NewServeMux()

	// Static files (index.html, img/*, etc.)
	s.Handle("/", http.FileServer(http.Dir("www")))

	s.HandleFunc("/_ah/health", healthCheckHandler)

	s.HandleFunc("/ip/", handleIPQuery)
	s.HandleFunc("/autnum/", handleAutnumQuery)
	s.HandleFunc("/domain/", handleDomainQuery)
	s.HandleFunc("/help", handleHelpQuery)

	s.ServeHTTP(w, r)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}

func handleHelpQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/rdap+json")

	helpJSON := `{
	"rdapConformance" :
	[
		"rdap_level_0"
	],
	"notices" :
	[
		{
		"title" : "www.rdap.net help",
		"description" :
		[
			"www.rdap.net provides a free RDAP query service."
		],
		"links" :
		[
			{
			"value" : "https://www.openrdap.org/api",
			"rel" : "alternate",
			"type" : "text/html",
			"href" : "https://www.openrdap.org/api"
			}
		]
		}
	]
}`

	fmt.Fprintf(w, "%s", helpJSON)
}

func handleIPQuery(w http.ResponseWriter, r *http.Request) {
	autnum := strings.TrimPrefix(r.URL.Path, "/ip/")

	regType := bootstrap.IPv4
	if strings.ContainsAny(autnum, ":") {
		regType = bootstrap.IPv6
	}

	processBootstrappedQuery(w, r, regType, autnum)
}

func handleDomainQuery(w http.ResponseWriter, r *http.Request) {
	domain := strings.TrimPrefix(r.URL.Path, "/domain/")
	processBootstrappedQuery(w, r, bootstrap.DNS, domain)
}

func handleAutnumQuery(w http.ResponseWriter, r *http.Request) {
	autnum := strings.TrimPrefix(r.URL.Path, "/autnum/")
	processBootstrappedQuery(w, r, bootstrap.ASN, autnum)
}

func processBootstrappedQuery(w http.ResponseWriter, r *http.Request, regType bootstrap.RegistryType, query string) {
	ctx := appengine.NewContext(r)

	// Empty query?
	if query == "" {
		http.NotFound(w, r)
		return
	}

	// Question for bootstrap client.
	question := &bootstrap.Question{
		RegistryType: regType,
		Query:        query,
	}
	question = question.WithContext(ctx)

	// Bootstrap client.
	client := &bootstrap.Client{
		HTTP:    urlfetch.Client(ctx),
		Cache:   caching.NewMemcacheCache(ctx),
		Verbose: func(s string) { fmt.Fprintf(os.Stderr, "%s\n", s) },
	}

	// Lookup question.
	answer, err := client.Lookup(question)

	// Not found?
	if err != nil || len(answer.URLs) == 0 {
		http.NotFound(w, r)
		return
	}

	// Build redirect URL.
	u := answer.URLs[0]
	u.Path = strings.TrimRight(u.Path, "/")
	u.Path += r.URL.Path
	u.RawQuery = r.URL.RawQuery

	// Redirect...
	http.Redirect(w, r, u.String(), 302)
}
