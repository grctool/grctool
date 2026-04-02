---
title: "AccountableHQ Scraping Library Evaluation"
phase: "02-design"
category: "design-reference"
status: "Complete"
tags: ["accountablehq", "scraping", "evaluation"]
related: ["FEAT-001", "SD-001"]
historical_aliases: ["SD-001-scraping-evaluation"]
created: 2026-03-18
updated: 2026-04-01
---

# Go Scraping Library Evaluation for AccountableHQ

This reference was previously filed as `SD-001-scraping-evaluation`. It
captures supporting implementation research for
[SD-001: AccountableHQ Integration
Adapter](/home/erik/Projects/grctool/docs/helix/02-design/solution-designs/SD-001-accountablehq-adapter.md)
and is not a standalone solution design.

## Libraries Evaluated

### 1. colly (github.com/gocolly/colly)

**Type**: HTTP-based scraper (no JavaScript execution)
**Maturity**: 23k+ GitHub stars, well-maintained
**Strengths**:
- Fast — direct HTTP, no browser overhead
- Built-in rate limiting, caching, cookie handling
- Elegant callback-based API (`OnHTML`, `OnRequest`)
- Robots.txt compliance
- Proxy support

**Weaknesses**:
- Cannot execute JavaScript — fails on SPAs (React/Angular/Vue)
- No dynamic content rendering

**Use when**: Target is server-rendered HTML

### 2. chromedp (github.com/chromedp/chromedp)

**Type**: Chrome DevTools Protocol (headless Chrome)
**Maturity**: 11k+ stars, actively maintained
**Strengths**:
- Full JavaScript execution — works on any SPA
- Real browser rendering (handles CSRF, dynamic tokens)
- Screenshot/PDF generation
- Network interception (can capture API calls the SPA makes)
- Headless mode for CI

**Weaknesses**:
- Requires Chrome/Chromium installed
- Slower than HTTP scraping (browser startup, rendering)
- More complex session management
- Higher memory usage

**Use when**: Target is a SPA or requires JS execution

### 3. rod (github.com/go-rod/rod)

**Type**: Chrome DevTools Protocol (headless Chrome)
**Maturity**: 5k+ stars, actively maintained
**Strengths**:
- Auto-downloads Chromium (no pre-install needed)
- Simpler API than chromedp
- Built-in element waiting, screenshots
- Stealth mode (anti-detection)

**Weaknesses**:
- Same browser overhead as chromedp
- Slightly less mature ecosystem
- Auto-download may conflict with CI lockdown policies

**Use when**: Want headless Chrome with simpler API

### 4. goquery (github.com/PuerkeBirl/goquery)

**Type**: HTML parser (jQuery-like selectors on static HTML)
**Maturity**: 14k+ stars, stable
**Strengths**:
- Fast HTML parsing with CSS selectors
- Works with any HTTP client (pair with colly or net/http)
- No browser required

**Weaknesses**:
- Not a scraper — just a parser. Needs separate HTTP client.
- No JavaScript execution

**Use when**: Need to parse HTML from any source

## Recommendation Matrix

| Criterion | colly | chromedp | rod | goquery |
|-----------|-------|----------|-----|---------|
| SPA support | No | Yes | Yes | No |
| Speed | Fast | Slow | Slow | N/A (parser) |
| Session/cookie mgmt | Built-in | Manual | Built-in | N/A |
| JS execution | No | Yes | Yes | No |
| CI compatibility | Easy | Needs Chrome | Auto-download | Easy |
| Rate limiting | Built-in | Manual | Manual | N/A |
| API call interception | No | Yes | Yes | No |
| Go module maturity | High | High | Medium | High |

## Recommended Approach

**Two-phase strategy:**

### Phase 1: Network Interception (chromedp/rod)
Before scraping HTML, use headless Chrome to **intercept XHR/fetch calls** that AccountableHQ's web UI makes. Most SPAs have an internal REST or GraphQL API — if we can discover it, we get structured JSON instead of parsing HTML.

This is the highest-value approach:
- Structured data (JSON) instead of fragile HTML parsing
- API contracts are more stable than UI layouts
- Faster to implement and maintain

### Phase 2: HTML Scraping (colly + goquery or chromedp)
If no usable internal API is found:
- **Server-rendered pages**: colly + goquery (fast, simple)
- **SPA pages**: chromedp or rod (slower but handles any JS)

## Architecture: Scraper Abstraction

Regardless of library choice, the scraper implements our existing
`AccountableHQClient` interface (from internal/providers/accountablehq/provider.go).
The interface is already library-agnostic:

```go
type AccountableHQClient interface {
    TestConnection(ctx context.Context) error
    ListPolicies(ctx context.Context, page, pageSize int) ([]AHQPolicy, int, error)
    GetPolicy(ctx context.Context, id string) (*AHQPolicy, error)
    CreatePolicy(ctx context.Context, policy *AHQPolicy) (string, error)
    UpdatePolicy(ctx context.Context, id string, policy *AHQPolicy) error
    DeletePolicy(ctx context.Context, id string) error
}
```

The scraper is just another implementation of this interface — alongside
the existing `HTTPClient`. The provider doesn't know or care whether
the data comes from REST API calls or scraped HTML.

## Decision

**Recommended library: chromedp** (or rod as backup)

Rationale:
- We don't yet know if AccountableHQ is a SPA — headless Chrome handles both cases
- Network interception gives us the best chance of finding an internal API
- chromedp is the most mature Go headless Chrome library
- Falls back to HTML scraping if no API is found

**Do NOT add the dependency yet.** Wait until grct-9qj.1 (investigate AccountableHQ web stack) confirms what we're dealing with.
