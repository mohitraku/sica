package knowledge

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type IngestResult struct {
	Title     string
	Body      string
	SourceURL string
}

func IngestURL(rawURL string) (*IngestResult, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title := ""
	if t := doc.Find("title").First(); t.Length() > 0 {
		title = strings.TrimSpace(t.Text())
	}
	if title == "" {
		if t := doc.Find("h1").First(); t.Length() > 0 {
			title = strings.TrimSpace(t.Text())
		}
	}
	if title == "" {
		title = rawURL
	}

	doc.Find("script, style, nav, footer, header, aside, .sidebar, .nav, .footer, .header, .menu, .ads, .comments").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	var bodyParts []string

	article := doc.Find("article").First()
	if article.Length() > 0 {
		article.Find("p, h1, h2, h3, h4, h5, h6, ul, ol, blockquote, pre, code, table").Each(func(i int, s *goquery.Selection) {
			bodyParts = append(bodyParts, elementToMarkdown(s))
		})
	}

	if len(bodyParts) == 0 {
		doc.Find("p, h1, h2, h3, h4, h5, h6, ul, ol, blockquote, pre, code").Each(func(i int, s *goquery.Selection) {
			bodyParts = append(bodyParts, elementToMarkdown(s))
		})
	}

	if len(bodyParts) == 0 {
		bodyParts = append(bodyParts, doc.Find("body").Text())
	}

	return &IngestResult{
		Title:     title,
		Body:      strings.Join(bodyParts, "\n\n"),
		SourceURL: rawURL,
	}, nil
}

func elementToMarkdown(s *goquery.Selection) string {
	tag := goquery.NodeName(s)
	text := strings.TrimSpace(s.Text())
	if text == "" {
		return ""
	}

	switch tag {
	case "h1":
		return "# " + text
	case "h2":
		return "## " + text
	case "h3":
		return "### " + text
	case "h4":
		return "#### " + text
	case "h5":
		return "##### " + text
	case "h6":
		return "###### " + text
	case "blockquote":
		lines := strings.Split(text, "\n")
		for i, l := range lines {
			lines[i] = "> " + l
		}
		return strings.Join(lines, "\n")
	case "pre", "code":
		return "```\n" + text + "\n```"
	case "ul", "ol":
		var items []string
		s.Find("li").Each(func(i int, li *goquery.Selection) {
			items = append(items, "- "+strings.TrimSpace(li.Text()))
		})
		return strings.Join(items, "\n")
	case "table":
		return text
	default:
		return text
	}
}
