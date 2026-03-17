# Simple Frontend Demo

A minimal chat interface for the AI Backend — pure HTML, CSS, and vanilla JS with no build step.

## Prerequisites

The AI backend must be running on `http://localhost:8080`. See the root README for setup instructions.

## Running

Serve the files with any static file server:

```bash
# Python
python3 -m http.server 3000

# Node
npx serve -p 3000

# Or use Docker Compose (from the examples/ directory)
docker compose up
```

Then open [http://localhost:3000](http://localhost:3000) in your browser.

## Features

- Non-streaming and streaming (SSE) chat
- Model selector (defaults to `openai/gpt-4o-mini`)
- Token usage tracking
- Conversation history loaded on page refresh
- Dark theme, responsive layout
