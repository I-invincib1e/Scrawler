package fetch

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// NewHTTPClient returns an http.Client with the specified timeout seconds.
func NewHTTPClient(timeoutSecs int) *http.Client {
	return &http.Client{Timeout: time.Duration(timeoutSecs) * time.Second}
}

// FetchDocument fetches a URL and returns the parsed goquery document, raw bytes, and content-type.
func FetchDocument(client *http.Client, targetURL string, userAgent string) (*goquery.Document, []byte, string, error) {
	if !RobotsAllowed(client, targetURL, userAgent) {
		return nil, nil, "", &RobotsBlockedError{URL: targetURL}
	}
	// Per-host rate limiting
	if getMinDelay() > 0 {
		if u, err := url.Parse(targetURL); err == nil {
			throttle(u.Scheme + "://" + u.Host)
		}
	}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, nil, "", err
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, "", err
	}
	defer func(body io.ReadCloser) { _ = body.Close() }(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, resp.Header.Get("Content-Type"), err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, nil, resp.Header.Get("Content-Type"), err
	}
	return doc, data, resp.Header.Get("Content-Type"), nil
}

// Robots parsing and cache
type RobotsBlockedError struct{ URL string }

func (e *RobotsBlockedError) Error() string { return "blocked by robots.txt: " + e.URL }

 // robotsCache caches robots.txt per scheme+host to avoid re-fetching every request.
var robotsCache = make(map[string]*robotsTxt)
var robotsMu sync.Mutex

type robotsTxt struct {
	uaRules map[string][]robotRule // by lowercase UA, "*" for default
}
type robotRule struct {
	allow bool
	path  string
}

// RobotsAllowed checks if URL is allowed for the given user-agent.
func RobotsAllowed(client *http.Client, rawURL, userAgent string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := u.Scheme + "://" + u.Host
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		ua = "*"
	}

	// Fast-path cache lookup without holding network calls under the lock (double-checked locking)
	robotsMu.Lock()
	rob := robotsCache[host]
	robotsMu.Unlock()
	if rob == nil {
		fetched := fetchRobots(client, host, userAgent)
		robotsMu.Lock()
		if robotsCache[host] == nil {
			robotsCache[host] = fetched
		}
		rob = robotsCache[host]
		robotsMu.Unlock()
	}
	return rob.isAllowed(ua, u.EscapedPath())
}

func fetchRobots(client *http.Client, host, userAgent string) *robotsTxt {
	// default allow if fetch fails
	rob := &robotsTxt{uaRules: map[string][]robotRule{"*": {}}}
	req, _ := http.NewRequest(http.MethodGet, host+"/robots.txt", nil)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		return rob
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return rob
	}
	parseRobots(string(data), rob)
	return rob
}

func parseRobots(text string, rob *robotsTxt) {
	var currentUA []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		parts := strings.SplitN(l, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		switch key {
		case "user-agent":
			ua := strings.ToLower(val)
			currentUA = []string{ua}
			if _, ok := rob.uaRules[ua]; !ok {
				rob.uaRules[ua] = nil
			}
		case "allow":
			for _, ua := range currentUA {
				rob.uaRules[ua] = append(rob.uaRules[ua], robotRule{allow: true, path: val})
			}
			rob.uaRules["*"] = append(rob.uaRules["*"], robotRule{allow: true, path: val})
		case "disallow":
			for _, ua := range currentUA {
				rob.uaRules[ua] = append(rob.uaRules[ua], robotRule{allow: false, path: val})
			}
			rob.uaRules["*"] = append(rob.uaRules["*"], robotRule{allow: false, path: val})
		}
	}
}

func (r *robotsTxt) isAllowed(userAgent string, path string) bool {
	// very simple longest-match: disallow beats allow on equal prefix length
	best := 0
	allowed := true
	check := func(rules []robotRule) {
		for _, rule := range rules {
			p := rule.path
			if p == "" {
				continue
			}
			// Match only exact path or subpaths (boundary-aware), e.g., "/jobs" or "/jobs/...".
			if path == p || strings.HasPrefix(path, strings.TrimRight(p, "/")+"/") {
				plen := len(p)
				if plen > best {
					best = plen
					allowed = rule.allow
				} else if plen == best && allowed && !rule.allow {
					// On equal prefix length, Disallow wins over Allow
					allowed = false
				}
			}
		}
	}
	if rules, ok := r.uaRules[userAgent]; ok {
		check(rules)
	}
	check(r.uaRules["*"])
	return allowed
}

// Simple per-host rate limiter
var minDelay time.Duration
var hostLast = make(map[string]time.Time)
var hostMu sync.Mutex

func SetMinDelay(d time.Duration) { minDelay = d }
func getMinDelay() time.Duration  { return minDelay }

func throttle(host string) {
	if minDelay <= 0 {
		return
	}
	hostMu.Lock()
	last := hostLast[host]
	now := time.Now()
	if last.IsZero() || now.Sub(last) >= minDelay {
		hostLast[host] = now
		hostMu.Unlock()
		return
	}
	wait := minDelay - now.Sub(last)
	hostMu.Unlock()
	time.Sleep(wait)
	hostMu.Lock()
	hostLast[host] = time.Now()
	hostMu.Unlock()
}
