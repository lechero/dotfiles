package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

// adfNode is a node of the Atlassian Document Format, Jira's JSON for rich
// text in descriptions and comments.
type adfNode struct {
	Type  string
	Text  string
	Attrs map[string]any
	Marks []struct {
		Type  string
		Attrs map[string]any
	}
	Content []adfNode
}

// richText reads a description or comment body: ADF from Jira's v3 API, or
// a plain string from older ones.
func richText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var doc adfNode
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return strings.TrimSpace(markdown(doc))
}

// markdown turns an ADF document into Markdown. Nodes it doesn't know keep
// their text.
func markdown(doc adfNode) string {
	var b strings.Builder
	blocks(&b, doc.Content, "")
	return b.String()
}

// blocks writes block nodes, each line starting with indent.
func blocks(b *strings.Builder, nodes []adfNode, indent string) {
	for _, n := range nodes {
		switch n.Type {
		case "paragraph":
			b.WriteString(indent + inline(n.Content) + "\n\n")
		case "heading":
			level := 1
			if l, ok := n.Attrs["level"].(float64); ok {
				level = int(l)
			}
			b.WriteString(indent + strings.Repeat("#", level) + " " + inline(n.Content) + "\n\n")
		case "bulletList", "orderedList":
			for i, item := range n.Content {
				marker := "- "
				if n.Type == "orderedList" {
					marker = fmt.Sprintf("%d. ", i+1)
				}
				var sub strings.Builder
				blocks(&sub, item.Content, "")
				lines := strings.Split(strings.TrimRight(sub.String(), "\n"), "\n")
				for j, line := range lines {
					if line == "" {
						continue
					}
					if j == 0 {
						b.WriteString(indent + marker + line + "\n")
					} else {
						b.WriteString(indent + strings.Repeat(" ", len(marker)) + line + "\n")
					}
				}
			}
			b.WriteString("\n")
		case "codeBlock":
			lang, _ := n.Attrs["language"].(string)
			b.WriteString(indent + "```" + lang + "\n" + plain(n.Content) + "\n" + indent + "```\n\n")
		case "blockquote":
			var sub strings.Builder
			blocks(&sub, n.Content, "")
			for _, line := range strings.Split(strings.TrimRight(sub.String(), "\n"), "\n") {
				b.WriteString(indent + "> " + line + "\n")
			}
			b.WriteString("\n")
		case "rule":
			b.WriteString(indent + "---\n\n")
		case "panel", "expand", "nestedExpand", "layoutSection", "layoutColumn", "taskList", "decisionList":
			blocks(b, n.Content, indent)
		case "taskItem", "decisionItem":
			box := "- [ ] "
			if state, _ := n.Attrs["state"].(string); state == "DONE" || state == "DECIDED" {
				box = "- [x] "
			}
			b.WriteString(indent + box + inline(n.Content) + "\n")
		case "table":
			table(b, n, indent)
		case "mediaSingle", "mediaGroup", "media":
			b.WriteString(indent + "*(attachment)*\n\n")
		default:
			if text := inline(n.Content) + n.Text; text != "" {
				b.WriteString(indent + text + "\n\n")
			}
		}
	}
}

func table(b *strings.Builder, n adfNode, indent string) {
	for i, row := range n.Content {
		cells := make([]string, len(row.Content))
		for j, cell := range row.Content {
			var sub strings.Builder
			blocks(&sub, cell.Content, "")
			cells[j] = strings.ReplaceAll(strings.TrimSpace(sub.String()), "\n", " ")
		}
		b.WriteString(indent + "| " + strings.Join(cells, " | ") + " |\n")
		if i == 0 {
			b.WriteString(indent + strings.Repeat("| --- ", len(cells)) + "|\n")
		}
	}
	b.WriteString("\n")
}

// inline writes text nodes and their marks.
func inline(nodes []adfNode) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			b.WriteString(marked(n))
		case "hardBreak":
			b.WriteString("  \n")
		case "mention", "emoji", "status", "date":
			b.WriteString(attrText(n))
		case "inlineCard", "blockCard", "embedCard":
			url, _ := n.Attrs["url"].(string)
			b.WriteString(url)
		default:
			b.WriteString(inline(n.Content))
		}
	}
	return b.String()
}

func marked(n adfNode) string {
	text := n.Text
	for _, m := range n.Marks {
		switch m.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "link":
			if href, ok := m.Attrs["href"].(string); ok {
				text = "[" + text + "](" + href + ")"
			}
		}
	}
	return text
}

// attrText is how an inline node like a mention or an emoji reads.
func attrText(n adfNode) string {
	for _, key := range []string{"text", "shortName"} {
		if s, ok := n.Attrs[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// plain is the bare text of nodes, for code blocks.
func plain(nodes []adfNode) string {
	var b strings.Builder
	for _, n := range nodes {
		b.WriteString(n.Text)
		b.WriteString(plain(n.Content))
	}
	return b.String()
}
