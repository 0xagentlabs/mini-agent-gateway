---
name: web-search
description: Search the web using DuckDuckGo or other search engines
metadata:
  openclaw:
    requires:
      bins: ["curl"]
---

## When to use

Use this skill when the user asks about:
- Current events or news
- Specific facts that may not be in your training data
- Technical documentation
- Product information

## How to search

Use the `fs:exec` tool with curl:

```bash
curl -s "https://html.duckduckgo.com/html/?q={encoded_query}"
```

Then parse the HTML to extract relevant results.

## Best practices

1. **Encode the query**: URL-encode special characters
2. **Limit results**: Get top 3-5 results
3. **Summarize**: Provide concise summaries, not raw HTML
4. **Cite sources**: Include URLs for verification
5. **Check dates**: Prefer recent results for time-sensitive topics

## Example

User: "What's the latest version of Go?"

You should:
1. Search: `curl -s "https://html.duckduckgo.com/html/?q=go+golang+latest+version+2025"`
2. Parse results
3. Answer: "The latest stable version of Go is 1.23..."
