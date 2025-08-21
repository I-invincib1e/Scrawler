# 🕷️ Scraper — Cobra-powered, Colorful Web Scraper CLI

[![Go](https://img.shields.io/badge/Go-1.25-blue?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
![CLI](https://img.shields.io/badge/CLI-cobra-7F52FF)
![Logs](https://img.shields.io/badge/logs-fatih%2Fcolor-green)
![OS](https://img.shields.io/badge/OS-Windows%20%7C%20Linux-blue)
![Status](https://img.shields.io/badge/Status-Active-brightgreen)

A fast, readable, and extensible web scraper CLI built with spf13/cobra and fatih/color.
- 🚀 Commands: `scraper crawl`, `scraper test robots`
- 🎛️ Short flags: `-u` (url), `-d` (depth), `-o` (out)
- 🎨 Colorized output for success, error, and warnings
- 🧠 Features: concurrency, depth control, extraction, robots.txt checks

## ✨ Features
- ✅ Cobra-based CLI with auto-generated `--help`
- ✅ Short flags: `-u`, `-d`, `-o`
- ✅ Concurrency control and rate limiting
- ✅ Optional extraction with pluggable output formats
- ✅ Robots.txt allow/deny tester
- ✅ Colorful, readable logs via `fatih/color`

## ⚙️ Install
Build the CLI:
```
go build -o scraper.exe .
```

Check it runs:
```
.\scraper.exe --help
```

## 🧭 Usage

Root help:
```
scraper --help
```

Crawl a site with short flags:
```
scraper crawl -u https://example.com -d 2 -o out
```

Crawl with concurrency and extraction:
```
scraper crawl -u https://example.com --concurrency 5 --max-pages 100 --extract --save-extract --format json --extract-save-format json -o out_extract
```

Test robots.txt rules:
```
scraper test robots -u https://python.org/ --user-agent "MyBot/1.0"
```

## 🖨️ Example Output (Colorized)
```
🚀 Starting crawler...
✓ Found 1 URLs to crawl
🌐 Crawling: https://example.com
🔄 Using sequential crawling
✓ Saved (1): https://example.com
🎉 Crawl complete. Fetched 1 page(s)
```

Robots test:
```
🧪 Testing robots.txt for: https://python.org/
📡 Fetching robots.txt...
✘ Robots.txt blocks access to https://python.org/ for user agent 'MyBot/1.0'
```

## 📦 Project Layout
- `cmd/` — Cobra commands (`root`, `crawl`, `test`)
- `scraper/` — Core logic (fetch, crawl, parse, output, util)
- `main.go` — Entrypoint delegating to Cobra

## 📝 License
This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.

## 🤝 Contributing
PRs welcome! Open an issue or submit a PR to improve commands, flags, or docs.
