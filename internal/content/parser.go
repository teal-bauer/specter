package content

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

// Frontmatter holds post/page metadata from markdown frontmatter
type Frontmatter struct {
	Title       string   `yaml:"title"`
	Slug        string   `yaml:"slug"`
	Tags        []string `yaml:"tags"`
	Featured    bool     `yaml:"featured"`
	Status      string   `yaml:"status"`
	Excerpt     string   `yaml:"excerpt"`
	MetaTitle   string   `yaml:"meta_title"`
	MetaDesc    string   `yaml:"meta_description"`
	FeatureImg  string   `yaml:"feature_image"`
	PublishedAt string   `yaml:"published_at"`
}

// ParsedContent contains parsed frontmatter and HTML content
type ParsedContent struct {
	Frontmatter Frontmatter
	HTML        string
	Markdown    string
}

// ParseFile reads a markdown file with frontmatter
func ParseFile(path string) (*ParsedContent, error) {
	if path == "-" {
		return ParseReader(os.Stdin)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	return ParseReader(f)
}

// ParseReader parses markdown with frontmatter from a reader
func ParseReader(r io.Reader) (*ParsedContent, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	return Parse(data)
}

// Parse parses markdown content with YAML frontmatter
func Parse(data []byte) (*ParsedContent, error) {
	content := &ParsedContent{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	// Check for frontmatter
	if scanner.Scan() && strings.TrimSpace(scanner.Text()) == "---" {
		var frontmatterBuf bytes.Buffer
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) == "---" {
				break
			}
			frontmatterBuf.WriteString(line)
			frontmatterBuf.WriteString("\n")
		}

		if err := yaml.Unmarshal(frontmatterBuf.Bytes(), &content.Frontmatter); err != nil {
			return nil, fmt.Errorf("parsing frontmatter: %w", err)
		}
	}

	// Rest is markdown content
	var markdownBuf bytes.Buffer
	for scanner.Scan() {
		markdownBuf.WriteString(scanner.Text())
		markdownBuf.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning content: %w", err)
	}

	content.Markdown = markdownBuf.String()

	// Convert markdown to HTML
	var htmlBuf bytes.Buffer
	md := goldmark.New()
	if err := md.Convert(markdownBuf.Bytes(), &htmlBuf); err != nil {
		return nil, fmt.Errorf("converting markdown: %w", err)
	}
	content.HTML = htmlBuf.String()

	return content, nil
}
