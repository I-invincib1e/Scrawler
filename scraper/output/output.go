package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scrawler/scraper/parse"
)

func RenderMarkdown(sig parse.Signals) string {
	var b strings.Builder
	if sig.Title != "" {
		b.WriteString("# " + sig.Title + "\n\n")
	}
	if sig.MetaDesc != "" {
		b.WriteString("> " + sig.MetaDesc + "\n\n")
	}
	if len(sig.Headings) > 0 {
		b.WriteString("## Headings\n\n")
		for _, h := range sig.Headings {
			b.WriteString("- " + h + "\n")
		}
		b.WriteString("\n")
	}
	if len(sig.Paragraphs) > 0 {
		b.WriteString("## Paragraphs\n\n")
		for _, p := range sig.Paragraphs {
			b.WriteString(p + "\n\n")
		}
	}
	if len(sig.Links) > 0 {
		b.WriteString("## Links\n\n")
		for _, l := range sig.Links {
			b.WriteString("- " + l + "\n")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func RenderPlainText(sig parse.Signals) string {
	var b strings.Builder
	if sig.Title != "" {
		b.WriteString(sig.Title + "\n\n")
	}
	if sig.MetaDesc != "" {
		b.WriteString(sig.MetaDesc + "\n\n")
	}
	for _, h := range sig.Headings {
		b.WriteString(h + "\n")
	}
	if len(sig.Headings) > 0 {
		b.WriteString("\n")
	}
	for _, p := range sig.Paragraphs {
		b.WriteString(p + "\n\n")
	}
	for _, l := range sig.Links {
		b.WriteString(l + "\n")
	}
	if len(sig.Links) > 0 {
		b.WriteString("\n")
	}
	return b.String()
}

func RenderJSON(sig parse.Signals) ([]byte, error) {
	return json.MarshalIndent(sig, "", "  ")
}

func SaveExtraction(baseOutDir, relDir, fileBase, format string, sig parse.Signals) error {
	absDir := filepath.Join(baseOutDir, "extract", relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return err
	}
	switch strings.ToLower(format) {
	case "md", "markdown":
		return os.WriteFile(filepath.Join(absDir, fileBase+".md"), []byte(RenderMarkdown(sig)), 0o644)
	case "txt", "text":
		return os.WriteFile(filepath.Join(absDir, fileBase+".txt"), []byte(RenderPlainText(sig)), 0o644)
	default:
		data, err := RenderJSON(sig)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(absDir, fileBase+".json"), data, 0o644)
	}
}

// BuildRelativePath returns relDir and fileBase given host, path, and rawquery
func BuildRelativePath(host, escapedPath, rawQuery string) (string, string) {
	hostDir := sanitize(host)
	pathPart := escapedPath
	if pathPart == "" {
		pathPart = "/"
	}
	var fileBase string
	if strings.HasSuffix(pathPart, "/") {
		fileBase = "index"
	} else {
		base := filepath.Base(pathPart)
		if strings.Contains(base, ".") {
			base = strings.TrimSuffix(base, filepath.Ext(base))
		}
		fileBase = base
		pathPart = filepath.Dir(pathPart) + "/"
	}
	if rawQuery != "" {
		fileBase = fmt.Sprintf("%s-%x", fileBase, parseHash(rawQuery))
	}
	relDir := filepath.Join(hostDir, filepath.FromSlash(pathPart))
	return relDir, fileBase
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

func parseHash(s string) uint64 {
	var h uint64 = 1469598103934665603
	const prime64 = 1099511628211
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}
