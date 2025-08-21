package util

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func ConfigureLogging(verbose, silent bool) {
	log.SetFlags(0)
	if silent {
		log.SetOutput(io.Discard)
		return
	}
	if verbose {
		log.SetFlags(log.LstdFlags)
	}
}

func GatherURLs(singleURL, urlFile string) ([]string, error) {
	if urlFile == "" {
		return []string{singleURL}, nil
	}
	f, err := os.Open(urlFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var urls []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", urlFile)
	}
	return urls, nil
}
