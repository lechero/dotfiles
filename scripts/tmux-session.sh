#!/bin/bash

# Use current directory if no argument is passed
DIR="${1:-$PWD}"
SESSION_NAME=$(basename "$DIR")

# Does session exist already?
if tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
  echo "Session '$SESSION_NAME' already exists. Switching..."
else
  echo "Creating new session '$SESSION_NAME' in $DIR..."

  # Create session with a named first window
  tmux new-session -d -s "$SESSION_NAME" -c "$DIR" -n main

  # Give tmux a moment to register
  sleep 0.2

  # Split the named window "main" horizontally
  tmux split-window -v -t "$SESSION_NAME":main -c "$DIR"

  # Create a second window called "second"
  tmux new-window -t "$SESSION_NAME" -n second -c "$DIR"

  # Switch back to the first window (main) and first pane
  tmux select-window -t "$SESSION_NAME":main
  tmux select-pane -t "$SESSION_NAME":main.0
fi

# Attach or switch to the session
if [ -n "$TMUX" ]; then
  tmux switch-client -t "$SESSION_NAME"
else
  tmux attach-session -t "$SESSION_NAME"
fi
