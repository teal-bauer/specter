# specter

CLI for the [Ghost Admin API](https://ghost.org/docs/admin-api/). Manage your Ghost blog from the terminal.

## Installation

Download a binary from [Releases](https://github.com/teal-bauer/specter/releases), or build from source:

```bash
go install github.com/teal-bauer/specter@latest
```

## Quick Start

```bash
# Interactive setup - opens browser to create API integration
specter login

# List recent posts
specter posts list

# Create a post from markdown
specter posts create my-post.md

# Get site info
specter site info
```

## Configuration

Run `specter login` for interactive setup, or configure manually:

**Environment variables:**
```bash
export GHOST_URL=https://myblog.com
export GHOST_ADMIN_KEY=64xxxxx:xxxxxxxxxxxxxx
```

**Config file** (`~/.config/specter/config.yaml`):
```yaml
default: myblog
instances:
  myblog:
    url: https://myblog.com
    key: "64xxxxx:xxxxxxxxxxxxxx"
  work:
    url: https://work.ghost.io
    key: "65xxxxx:xxxxxxxxxxxxxx"
```

### Multiple Profiles

```bash
# Set up multiple Ghost sites
specter login myblog
specter login work

# Use a specific profile
specter -p work posts list

# Or via environment
GHOST_PROFILE=work specter posts list

# List configured profiles
specter profiles
```

## Commands

```
specter posts       list|get|create|update|delete
specter pages       list|get|create|update|delete
specter tags        list|get|create|update|delete
specter members     list|get|create|update|delete
specter tiers       list|get|create|update
specter newsletters list|get|create|update
specter images      upload
specter site        info
specter users       list|get
specter profiles    list configured profiles
specter login       interactive setup
```

### Global Flags

```
-p, --profile    Config profile to use
-o, --output     Output format: text or json (default "text")
    --url        Ghost site URL (override config)
    --key        Ghost Admin API key (override config)
```

## Shell Completion

```bash
# Zsh (add to ~/.zshrc)
source <(specter completion zsh)

# Bash (add to ~/.bashrc)
source <(specter completion bash)

# Fish (one-time)
specter completion fish > ~/.config/fish/completions/specter.fish
```

## Markdown Input

Posts and pages accept markdown files with YAML frontmatter:

```markdown
---
title: "My Post Title"
tags:
  - Technology
  - Go
slug: my-post-title
featured: true
status: draft
excerpt: "A short description"
feature_image: https://example.com/image.jpg
---

Post content here in markdown...
```

Create or update:

```bash
# Create as draft (default)
specter posts create my-post.md

# Create and publish immediately
specter posts create my-post.md --status published

# Schedule for later
specter posts create my-post.md --status scheduled --publish-at 2025-01-20T10:00:00Z

# Update existing post
specter posts update my-post-slug updated-content.md

# Read from stdin
cat post.md | specter posts create -
```

## JSON Output

Use `-o json` for scripting:

```bash
# Get post IDs
specter posts list -o json | jq '.[].id'

# Export all posts
specter posts list --all -o json > posts.json
```

## License

[GPL-3.0](LICENSE)
