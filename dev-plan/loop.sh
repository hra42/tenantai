#!/bin/bash

##############################################################################
# Orchestrate Phase 3 & 4 with Ralph Wiggum Plugin
# 
# Usage:
#   ./orchestrate-phases.sh [--phase 3|4|both] [--max-iter N] [--skip-checks]
#
# Examples:
#   ./orchestrate-phases.sh                    # Run both phases
#   ./orchestrate-phases.sh --phase 3          # Phase 3 only
#   ./orchestrate-phases.sh --max-iter 60      # Custom iteration limit
#
##############################################################################

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PHASE_TO_RUN="both"  # both, 3, 4
MAX_ITER_PHASE_3=50
MAX_ITER_PHASE_4=40
SKIP_CHECKS=false
VERBOSE=false

# Trap errors and cleanup
trap 'on_error' ERR
trap 'on_interrupt' INT TERM

on_error() {
    echo -e "${RED}❌ Error occurred. Check logs above.${NC}"
    exit 1
}

on_interrupt() {
    echo -e "${RED}\n⏹️  Interrupted by user${NC}"
    exit 130
}

##############################################################################
# Helper Functions
##############################################################################

log_section() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_step() {
    echo -e "${YELLOW}🔄 $1${NC}"
}

##############################################################################
# Parse Command Line Arguments
##############################################################################

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --phase)
                PHASE_TO_RUN="$2"
                if [[ ! "$PHASE_TO_RUN" =~ ^(both|3|4)$ ]]; then
                    log_error "Invalid phase: $PHASE_TO_RUN (must be 'both', '3', or '4')"
                    exit 1
                fi
                shift 2
                ;;
            --max-iter)
                # Set for both phases
                MAX_ITER_PHASE_3="$2"
                MAX_ITER_PHASE_4="$2"
                shift 2
                ;;
            --skip-checks)
                SKIP_CHECKS=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat << 'EOF'
Orchestrate Phase 3 & 4 with Ralph Wiggum Plugin

Usage:
  ./orchestrate-phases.sh [OPTIONS]

Options:
  --phase PHASE              Which phase(s) to run: 'both' (default), '3', or '4'
  --max-iter N               Set iteration limit for all phases (overrides defaults)
  --skip-checks              Skip pre-flight checks
  --verbose                  Enable verbose logging
  --help                     Show this help message

Examples:
  ./orchestrate-phases.sh                         # Run both phases with defaults
  ./orchestrate-phases.sh --phase 3               # Phase 3 only
  ./orchestrate-phases.sh --phase 4 --max-iter 60 # Phase 4 with 60 iterations
  ./orchestrate-phases.sh --verbose                # Verbose output

Environment Variables:
  OPENROUTER_API_KEY        Required for Phase 3 (OpenRouter API key)
  DEBUG                      Set to '1' for debug output

EOF
}

##############################################################################
# Pre-flight Checks
##############################################################################

run_preflight_checks() {
    log_section "🔍 Running Pre-flight Checks"

    # Check if we're in a Git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a Git repository. Initialize with: git init"
        exit 1
    fi
    log_success "Git repository found"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go not found. Install Go 1.22+ to continue."
        exit 1
    fi
    GO_VERSION=$(go version | sed 's/.*go\([0-9.]*\).*/\1/')
    log_success "Go $GO_VERSION found"

    # Check if PLAN.md exists
    if [[ ! -f "dev-plan/phases.md" ]]; then
        log_error "PLAN.md not found in current directory"
        exit 1
    fi
    log_success "PLAN.md found"

    # Check if go.mod exists (Phase 1 should have created it)
    if [[ ! -f "go.mod" ]]; then
        log_warning "go.mod not found. Phase 1 may not be complete."
        log_warning "You may need to initialize the project first."
    else
        log_success "go.mod found"
    fi

    # Check OPENROUTER_API_KEY if running Phase 3 or both
    if [[ "$PHASE_TO_RUN" == "both" ]] || [[ "$PHASE_TO_RUN" == "3" ]]; then
        if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
            log_warning "OPENROUTER_API_KEY not set"
            log_info "Phase 3 will need this. Set it with: export OPENROUTER_API_KEY=<your-key>"
        else
            log_success "OPENROUTER_API_KEY is set"
        fi
    fi

    # Check if Claude Code is available (Ralph plugin)
    if ! command -v ralph-loop &> /dev/null 2>/dev/null; then
        log_warning "Ralph plugin may not be available in PATH"
        log_info "Ensure Claude Code is installed and Ralph plugin is configured"
    fi

    log_success "Pre-flight checks complete"
    echo
}

##############################################################################
# Phase 3: Integration & Testing
##############################################################################

run_phase_3() {
    log_section "🚀 Phase 3: Integration & Testing"
    log_info "Max iterations: $MAX_ITER_PHASE_3"
    log_info "Completion phrase: PHASE_3_DONE"
    echo

    local prompt_3=$(cat <<'PHASE3_PROMPT'
Execute Phase 3: Integration & Testing from PLAN.md

Phase 3 consists of 4 subsections:

**3.1 OpenRouter SDK Integration**
- Review https://github.com/hra42/openrouter-go SDK
- Create thin openrouter/client.go wrapper
- Handle errors from OpenRouter (rate limits, invalid models, auth)
- Support streaming responses if SDK supports
- Add request/response logging in debug mode
- Document required OPENROUTER_API_KEY env var

**3.2 Unit Tests**
Critical paths to test:
- Service creation + DB initialization
- Middleware: service context injection (valid + invalid service IDs)
- Chat completion: valid request → logged to DB
- Error handling: malformed requests, missing service, OpenRouter errors
- Conversation queries: pagination, filters
Set up table-driven tests for edge cases.

**3.3 Integration Tests**
- Spin up real DuckDB in-memory for tests
- Full request flow: POST /v1/chat/completions → GET /services/{id}/conversations
- Test concurrent requests to same service (no DB conflicts)
- Test multi-service isolation

**3.4 Documentation**
- README.md: Quick start (clone → config.yaml + OPENROUTER_API_KEY → go run main.go)
- docs/API.md: Full endpoint reference with curl examples
- docs/ARCHITECTURE.md: Design decisions + data flow diagram
- docs/EXTENDING.md: Extension paths (auth, RAG, cost tracking, etc.)
- Inline code comments for non-obvious logic

**Success Criteria for Phase 3:**
- Unit tests: 70%+ coverage on core logic
- Integration tests: E2E flow passes
- Documentation complete + clear

Run tests to verify completion. Output <promise>PHASE_3_DONE</promise> when all tests pass and documentation is complete.
PHASE3_PROMPT
    )

    /ralph-loop "$prompt_3" --max-iterations "$MAX_ITER_PHASE_3" --completion-promise "PHASE_3_DONE"
    local phase_3_exit=$?

    if [[ $phase_3_exit -eq 0 ]]; then
        log_success "Phase 3 completed successfully"
        return 0
    else
        log_error "Phase 3 failed or hit iteration limit"
        return 1
    fi
}

##############################################################################
# Phase 4: Examples & Demo
##############################################################################

run_phase_4() {
    log_section "🚀 Phase 4: Examples & Demo"
    log_info "Max iterations: $MAX_ITER_PHASE_4"
    log_info "Completion phrase: PHASE_4_DONE"
    echo

    local prompt_4=$(cat <<'PHASE4_PROMPT'
Execute Phase 4: Examples & Demo from PLAN.md

Phase 4 consists of 3 subsections:

**4.1 Simple Frontend Example**
Location: examples/simple-frontend/
Tech: HTML + Vanilla JS (no build step)
Features:
- Create service on load (or use pre-created service)
- Chat UI: message input, send button, conversation display
- Displays conversation history on load
- Shows token usage if OpenRouter includes it

Files to create:
- index.html - Chat interface with cyberpunk aesthetic
- app.js - API calls + DOM updates
- style.css - Basic styling (dark theme, tech vibes)
- README.md - How to run

Design should be polished and reflect cyberpunk aesthetic.

**4.2 Docker Compose Setup**
File: examples/docker-compose.yml
Services:
- tenantai: Go app + DuckDB
- simple-frontend: Static HTTP server

Volumes:
- Mount ./data/services for DuckDB persistence
- Mount .env for config

Create:
- docker-compose.yml
- Dockerfile for Go app (multi-stage build, Alpine base)
- Document: docker-compose up → visit http://localhost:3000
- Test locally to ensure everything works

**4.3 Quick Start Script**
File: scripts/quickstart.sh
Actions:
1. Check Go installed
2. Copy config.example.yaml → config.yaml
3. Prompt for OPENROUTER_API_KEY
4. Prompt for port (default 8080)
5. Run go run main.go

Make it cross-platform (check for bash availability).

**Success Criteria for Phase 4:**
- Simple frontend runs in docker-compose up
- Screenshots in README
- Quickstart takes <5 minutes

Test docker-compose setup end-to-end. Output <promise>PHASE_4_DONE</promise> when all examples work.
PHASE4_PROMPT
    )

    /ralph-loop "$prompt_4" --max-iterations "$MAX_ITER_PHASE_4" --completion-promise "PHASE_4_DONE"
    local phase_4_exit=$?

    if [[ $phase_4_exit -eq 0 ]]; then
        log_success "Phase 4 completed successfully"
        return 0
    else
        log_error "Phase 4 failed or hit iteration limit"
        return 1
    fi
}

##############################################################################
# Main Execution
##############################################################################

main() {
    log_section "🎬 Ralph Wiggum Phase Orchestrator"
    echo "Phase(s) to run: $PHASE_TO_RUN"
    echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
    echo

    # Parse arguments
    parse_args "$@"

    # Pre-flight checks (unless skipped)
    if [[ "$SKIP_CHECKS" != "true" ]]; then
        run_preflight_checks
    else
        log_warning "Skipping pre-flight checks (--skip-checks enabled)"
    fi

    # Track overall status
    local overall_success=true

    # Run Phase 3
    if [[ "$PHASE_TO_RUN" == "both" ]] || [[ "$PHASE_TO_RUN" == "3" ]]; then
        if ! run_phase_3; then
            overall_success=false
            if [[ "$PHASE_TO_RUN" == "3" ]]; then
                # If only Phase 3 was requested, exit here
                log_error "Phase 3 failed. Exiting."
                exit 1
            else
                # If both were requested, log warning but try Phase 4
                log_warning "Phase 3 failed, but continuing to Phase 4"
            fi
        fi
        echo
    fi

    # Run Phase 4
    if [[ "$PHASE_TO_RUN" == "both" ]] || [[ "$PHASE_TO_RUN" == "4" ]]; then
        if ! run_phase_4; then
            overall_success=false
            log_error "Phase 4 failed."
            exit 1
        fi
        echo
    fi

    # Final Summary
    if [[ "$overall_success" == "true" ]]; then
        log_section "🎉 All Phases Completed Successfully!"
        echo "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
        log_success "tenantai project is ready for Phase 5: Production Readiness"
        echo
        log_info "Next steps:"
        echo "  1. Review Phase 3 & 4 outputs"
        echo "  2. Test docker-compose setup: docker-compose up"
        echo "  3. Visit http://localhost:3000 for the frontend"
        echo "  4. Proceed to Phase 5 when ready"
        echo
        exit 0
    else
        log_error "One or more phases failed. Review logs above."
        exit 1
    fi
}

##############################################################################
# Entry Point
##############################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
