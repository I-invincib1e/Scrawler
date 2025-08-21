package cmd

import (
	"fmt"

	"scrawler/scraper/fetch"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	testURL       string
	testUserAgent string
	testTimeout   int
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test robots.txt compliance",
	Long: `Test robots.txt compliance for a given URL.
This will fetch and parse the robots.txt file and check if the URL can be accessed by the specified user agent.`,
	Example: `  scraper test robots https://python.org/
  scraper test robots --user-agent "MyBot/1.0" https://example.com`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		testType := args[0]

		switch testType {
		case "robots":
			testRobotsTxt()
		default:
			color.Red("✘ Unknown test type: %s", testType)
			color.Yellow("Available test types: robots")
			return
		}
	},
}

func testRobotsTxt() {
	color.Cyan("🧪 Testing robots.txt for: %s", testURL)

	client := fetch.NewHTTPClient(testTimeout)

	color.Blue("📡 Fetching robots.txt...")

	// Test robots.txt compliance
	allowed := fetch.RobotsAllowed(client, testURL, testUserAgent)

	if allowed {
		color.Green("✓ Robots.txt allows access to %s for user agent '%s'", testURL, testUserAgent)
	} else {
		color.Red("✘ Robots.txt blocks access to %s for user agent '%s'", testURL, testUserAgent)
	}

	// Try to fetch robots.txt content to show more details
	robotsURL := testURL + "/robots.txt"
	color.Blue("📄 Fetching robots.txt content from: %s", robotsURL)

	_, rawBytes, contentType, err := fetch.FetchDocument(client, robotsURL, testUserAgent)
	if err != nil {
		color.Yellow("⚠ Warning: Could not fetch robots.txt: %s", err)
		return
	}

	if contentType != "" && !contains(contentType, "text/plain") {
		color.Yellow("⚠ Warning: robots.txt content type is %s, expected text/plain", contentType)
	}

	color.Green("✓ robots.txt content retrieved successfully")
	fmt.Println("--- robots.txt content ---")
	fmt.Println(string(rawBytes))
	fmt.Println("--- end robots.txt ---")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&testURL, "url", "u", "", "URL to test robots.txt for (required)")
	testCmd.Flags().StringVarP(&testUserAgent, "user-agent", "", "scrawler/0.1 (+https://example.local)", "User-Agent string")
	testCmd.Flags().IntVarP(&testTimeout, "timeout", "", 15, "HTTP timeout in seconds")

	testCmd.MarkFlagRequired("url")
}
