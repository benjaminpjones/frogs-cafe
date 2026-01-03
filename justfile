set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

server_cmd := "docker compose -f docker-compose.dev.yml up"
client_cmd := "cd web_client && npm run dev"

_default:
	@just --list

# Install all dependencies (Go modules and npm packages)
install:
	cd server && go mod download
	cd web_client && npm install

# Run server and client in split tmux session with hot reload
run:
	#!/usr/bin/env bash
	if ! command -v tmux >/dev/null 2>&1; then
	    echo "tmux is not installed."
	    echo "Install instructions: https://github.com/tmux/tmux/wiki/Installing"
	    echo "Or run the services separately:"
	    echo "  {{ server_cmd }}"
	    echo "  {{ client_cmd }}"
	    exit 1
	fi
	
	session="frogs_cafe_dev"
	if tmux has-session -t "$session" 2>/dev/null; then
	    tmux kill-session -t "$session"
	fi
	
	tmux new-session -d -s "$session" "{{ server_cmd }}"
	tmux split-window -h -t "$session" "{{ client_cmd }}"
	tmux select-layout -t "$session" even-horizontal
	tmux attach -t "$session"
