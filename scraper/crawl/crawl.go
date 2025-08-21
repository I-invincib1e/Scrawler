package crawl

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"scrawler/scraper/fetch"
	"scrawler/scraper/output"
	"scrawler/scraper/parse"

	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
)

type Options struct {
	StartURL          string
	UserAgent         string
	TimeoutSecs       int
	MaxDepth          int
	MaxPages          int
	SameHostOnly      bool
	OutDir            string
	Concurrency       int
	SaveExtract       bool
	ExtractSaveFormat string
}

func Crawl(opts Options) error {
	start, err := url.Parse(opts.StartURL)
	if err != nil {
		return err
	}
	client := fetch.NewHTTPClient(opts.TimeoutSecs)
	visited := make(map[string]bool)
	type qi struct {
		u     *url.URL
		depth int
	}
	queue := []qi{{u: start, depth: 0}}
	pages := 0

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		can := canonicalURL(item.u)
		if visited[can] {
			continue
		}
		if opts.MaxPages > 0 && pages >= opts.MaxPages {
			break
		}
		visited[can] = true

		doc, body, ctype, err := fetch.FetchDocument(client, item.u.String(), opts.UserAgent)
		if err != nil || !strings.Contains(strings.ToLower(ctype), "text/html") {
			continue
		}

		if err := saveHTML(opts.OutDir, item.u, body); err == nil {
			pages++
			color.Green("✓ Saved (%d): %s", pages, item.u.String())
		}
		// content-hash dedupe: skip exploring links if we've seen identical content
		if seenContent(body) {
			continue
		}
		if opts.SaveExtract {
			sig := parse.ExtractSignals(doc, item.u.String())
			relDir, fileBase := buildRel(item.u)
			if err := output.SaveExtraction(opts.OutDir, relDir, fileBase, opts.ExtractSaveFormat, sig); err != nil {
				color.Yellow("⚠ Warning: Failed to save extraction for %s: %v", item.u.String(), err)
			}
		}

		if item.depth >= opts.MaxDepth {
			continue
		}
		for _, link := range extractLinks(doc, item.u) {
			if link.Scheme != "http" && link.Scheme != "https" {
				continue
			}
			if opts.SameHostOnly && !sameHost(start, link) {
				continue
			}
			queue = append(queue, qi{u: link, depth: item.depth + 1})
		}
	}
	color.Cyan("🎉 Crawl complete. Fetched %d page(s)", pages)
	return nil
}

func CrawlConcurrent(opts Options) error {
	start, err := url.Parse(opts.StartURL)
	if err != nil {
		return err
	}
	client := fetch.NewHTTPClient(opts.TimeoutSecs)

	type job struct {
		u     *url.URL
		depth int
	}
	visited := make(map[string]bool)
	visitedMu := make(chan struct{}, 1)
	queue := make(chan job, 2048)
	done := make(chan struct{})
	pagesCh := make(chan struct{}, opts.MaxPages+10)
	activeWorkers := make(chan struct{}, opts.Concurrency)

	queue <- job{u: start, depth: 0}

	worker := func() {
		activeWorkers <- struct{}{} // Mark worker as active
		defer func() { <-activeWorkers }() // Mark worker as inactive

		for j := range queue {
			if opts.MaxPages > 0 && len(pagesCh) >= opts.MaxPages {
				continue
			}
			can := canonicalURL(j.u)
			visitedMu <- struct{}{}
			if visited[can] {
				<-visitedMu
				continue
			}
			visited[can] = true
			<-visitedMu

			doc, body, ctype, err := fetch.FetchDocument(client, j.u.String(), opts.UserAgent)
			if err != nil || !strings.Contains(strings.ToLower(ctype), "text/html") {
				continue
			}

			if err := saveHTML(opts.OutDir, j.u, body); err == nil {
				pagesCh <- struct{}{}
				color.Green("✓ Saved (%d): %s", len(pagesCh), j.u.String())
			}
			// content-hash dedupe: skip enqueueing links if identical content already seen
			if seenContent(body) {
				continue
			}
			if opts.SaveExtract {
				sig := parse.ExtractSignals(doc, j.u.String())
				relDir, fileBase := buildRel(j.u)
				if err := output.SaveExtraction(opts.OutDir, relDir, fileBase, opts.ExtractSaveFormat, sig); err != nil {
					color.Yellow("⚠ Warning: Failed to save extraction for %s: %v", j.u.String(), err)
				}
			}

			if j.depth >= opts.MaxDepth {
				continue
			}
			for _, link := range extractLinks(doc, j.u) {
				if link.Scheme != "http" && link.Scheme != "https" {
					continue
				}
				if opts.SameHostOnly && !sameHost(start, link) {
					continue
				}
				if opts.MaxPages > 0 && len(pagesCh) >= opts.MaxPages {
					break
				}
				queue <- job{u: link, depth: j.depth + 1}
			}
		}
		done <- struct{}{}
	}

	workers := opts.Concurrency
	if workers < 1 {
		workers = 1
	}
	if workers > runtime.NumCPU()*4 {
		workers = runtime.NumCPU() * 4
	}
	for i := 0; i < workers; i++ {
		go worker()
	}

	// Improved queue closure logic
	go func() {
		if opts.MaxPages > 0 {
			// Close when max pages reached
			for {
				if len(pagesCh) >= opts.MaxPages {
					close(queue)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		} else {
			// Close after timeout OR when all workers are idle and queue is empty
			timeout := time.After(2 * time.Second)
			for {
				select {
				case <-timeout:
					close(queue)
					return
				default:
					// Check if all workers are idle and queue is empty
					if len(queue) == 0 && len(activeWorkers) == 0 {
						// Give a small grace period for any pending enqueues
						time.Sleep(100 * time.Millisecond)
						if len(queue) == 0 {
							close(queue)
							return
						}
					}
					time.Sleep(50 * time.Millisecond)
				}
			}
		}
	}()

	for i := 0; i < workers; i++ {
		<-done
	}
	color.Cyan("🎉 Crawl complete. Fetched %d page(s)", len(pagesCh))
	return nil
}

// helpers (temporary; move to util as needed)
func sameHost(a, b *url.URL) bool { return strings.EqualFold(a.Hostname(), b.Hostname()) }
func canonicalURL(u *url.URL) string {
	clone := *u
	clone.Fragment = ""
	if (clone.Scheme == "http" && clone.Port() == "80") || (clone.Scheme == "https" && clone.Port() == "443") {
		clone.Host = clone.Hostname()
	}
	if clone.Path == "" {
		clone.Path = "/"
	}
	return clone.String()
}
func extractLinks(doc *goquery.Document, base *url.URL) []*url.URL {
	var out []*url.URL
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" {
			return
		}
		u, err := base.Parse(href)
		if err != nil {
			return
		}
		u.Fragment = ""
		out = append(out, u)
	})
	return out
}
func saveHTML(outDir string, u *url.URL, data []byte) error {
	host := sanitize(u.Hostname())
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	var name string
	if strings.HasSuffix(p, "/") {
		name = "index.html"
	} else {
		b := filepath.Base(p)
		if !strings.Contains(b, ".") {
			b += ".html"
		}
		name = b
		p = filepath.Dir(p) + "/"
	}
	rel := filepath.Join(host, filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Join(outDir, rel), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, rel, name), data, 0o644)
}
func sanitize(s string) string {
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if s == "" {
		return "site"
	}
	return strings.NewReplacer(":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_").Replace(s)
}
func buildRel(u *url.URL) (string, string) {
	host := sanitize(u.Hostname())
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	var base string
	if strings.HasSuffix(p, "/") {
		base = "index"
	} else {
		b := filepath.Base(p)
		if strings.Contains(b, ".") {
			b = strings.TrimSuffix(b, filepath.Ext(b))
		}
		base = b
		p = filepath.Dir(p) + "/"
	}
	return filepath.Join(host, filepath.FromSlash(p)), base
}

// content hash dedupe
var seenHashes = make(map[uint64]struct{})

func seenContent(b []byte) bool {
	var h uint64 = 1469598103934665603
	const prime64 = 1099511628211
	for i := 0; i < len(b); i++ {
		h ^= uint64(b[i])
		h *= prime64
	}
	if _, ok := seenHashes[h]; ok {
		return true
	}
	seenHashes[h] = struct{}{}
	return false
}
