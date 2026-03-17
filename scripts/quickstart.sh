#!/usr/bin/env bash
set -euo pipefail

echo "=== AI Backend Quickstart ==="
echo

# Check Go
if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed. Install Go 1.26+ from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')
GO_MAJOR=$(echo "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(echo "$GO_VERSION" | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 26 ]; }; then
    echo "Error: Go 1.26+ required, found go${GO_VERSION}"
    exit 1
fi

echo "Go ${GO_VERSION} detected."

# Config file
if [ ! -f config.yaml ]; then
    if [ -f config.example.yaml ]; then
        cp config.example.yaml config.yaml
        echo "Created config.yaml from config.example.yaml"
    else
        echo "Error: config.example.yaml not found. Are you in the project root?"
        exit 1
    fi
else
    echo "config.yaml already exists, keeping it."
fi

# API key
if [ -z "${OPENROUTER_API_KEY:-}" ]; then
    echo
    read -rp "Enter your OpenRouter API key: " api_key
    if [ -z "$api_key" ]; then
        echo "Error: API key is required. Get one at https://openrouter.ai/keys"
        exit 1
    fi
    export OPENROUTER_API_KEY="$api_key"
    echo "OPENROUTER_API_KEY set for this session."
else
    echo "OPENROUTER_API_KEY already set."
fi

# Port
echo
read -rp "Port [8080]: " port
port="${port:-8080}"

# Update config port if non-default
if [ "$port" != "8080" ]; then
    if command -v sed &>/dev/null; then
        sed -i.bak "s/port: 8080/port: ${port}/" config.yaml && rm -f config.yaml.bak
        echo "Config updated to use port ${port}."
    fi
fi

# Run
echo
echo "Starting AI Backend on port ${port}..."
echo "Press Ctrl+C to stop."
echo
exec go run main.go
