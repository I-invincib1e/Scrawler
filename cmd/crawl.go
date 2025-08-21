package cmd

import (
	"log"
	"time"

	"scrawler/scraper/crawl"
	"scrawler/scraper/fetch"
	"scrawler/scraper/util"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	crawlURL         string
	crawlURLFile     string
	crawlUserAgent   string
	crawlTimeout     int
	crawlMaxDepth    int
	crawlMaxPages    int
	crawlSameHost    bool
	crawlOutDir      string
	crawlConcurrency int
	crawlVerbose     bool
	crawlSilent      bool
	crawlExtract     bool
	crawlFormat      string
	crawlSaveExtract bool
	crawlSaveFormat  string
	crawlDelay       time.Duration
)

var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "Crawl a website",
	Long: `Crawl a website starting from the specified URL.
Supports concurrent crawling, depth control, and various output options.`,
	Example: `  scraper crawl -u https://example.com -d 2 -o json
  scraper crawl -u https://example.com --max-pages 100 --concurrency 5`,
	Run: func(cmd *cobra.Command, args []string) {
		color.Cyan("🚀 Starting crawler...")

		util.ConfigureLogging(crawlVerbose, crawlSilent)
		fetch.SetMinDelay(crawlDelay)

		urls, err := util.GatherURLs(crawlURL, crawlURLFile)
		if err != nil {
			color.Red("✘ Error gathering URLs: %s", err)
			log.Fatal(err)
		}

		color.Green("✓ Found %d URLs to crawl", len(urls))

		for _, u := range urls {
			color.Yellow("🌐 Crawling: %s", u)

			copts := crawl.Options{
				StartURL:          u,
				UserAgent:         crawlUserAgent,
				TimeoutSecs:       crawlTimeout,
				MaxDepth:          crawlMaxDepth,
				MaxPages:          crawlMaxPages,
				SameHostOnly:      crawlSameHost,
				OutDir:            crawlOutDir,
				Concurrency:       crawlConcurrency,
				SaveExtract:       crawlSaveExtract,
				ExtractSaveFormat: crawlSaveFormat,
			}

			var err error
			if copts.Concurrency <= 1 {
				color.Blue("🔄 Using sequential crawling")
				err = crawl.Crawl(copts)
			} else {
				color.Blue("🔄 Using concurrent crawling with %d workers", copts.Concurrency)
				err = crawl.CrawlConcurrent(copts)
			}

			if err != nil {
				color.Red("✘ Error during crawling: %s", err)
			} else {
				color.Green("✓ Successfully crawled: %s", u)
			}
		}

		color.Cyan("🎉 Crawling completed!")
	},
}

func init() {
	rootCmd.AddCommand(crawlCmd)

	// Add flags with short versions
	crawlCmd.Flags().StringVarP(&crawlURL, "url", "u", "", "Target URL to crawl (required)")
	crawlCmd.Flags().StringVarP(&crawlURLFile, "url-file", "", "", "File with one URL per line")
	crawlCmd.Flags().StringVarP(&crawlUserAgent, "user-agent", "", "scrawler/0.1 (+https://example.local)", "User-Agent string")
	crawlCmd.Flags().IntVarP(&crawlTimeout, "timeout", "", 15, "HTTP timeout in seconds")
	crawlCmd.Flags().IntVarP(&crawlMaxDepth, "max-depth", "d", 2, "Maximum crawl depth")
	crawlCmd.Flags().IntVarP(&crawlMaxPages, "max-pages", "", 50, "Maximum number of pages to crawl")
	crawlCmd.Flags().BoolVarP(&crawlSameHost, "same-host", "", true, "Restrict crawling to same host only")
	crawlCmd.Flags().StringVarP(&crawlOutDir, "out", "o", "out", "Output directory for crawled data")
	crawlCmd.Flags().IntVarP(&crawlConcurrency, "concurrency", "", 1, "Number of concurrent workers")
	crawlCmd.Flags().BoolVarP(&crawlVerbose, "verbose", "v", false, "Enable verbose logging")
	crawlCmd.Flags().BoolVarP(&crawlSilent, "silent", "", false, "Disable all logging")
	crawlCmd.Flags().BoolVarP(&crawlExtract, "extract", "", false, "Extract signals during crawl")
	crawlCmd.Flags().StringVarP(&crawlFormat, "format", "", "json", "Output format: json|md|txt")
	crawlCmd.Flags().BoolVarP(&crawlSaveExtract, "save-extract", "", false, "Save extraction results during crawl")
	crawlCmd.Flags().StringVarP(&crawlSaveFormat, "extract-save-format", "", "json", "Format for saved extractions")
	crawlCmd.Flags().DurationVarP(&crawlDelay, "delay", "", 0, "Minimum delay between requests (e.g., 1s)")

	// Mark required flags
	crawlCmd.MarkFlagRequired("url")
}
