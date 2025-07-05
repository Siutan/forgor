#!/bin/bash
set -e

SNIPPET_DIR="$HOME/shell-logger/logger-snippets"

cleanup_file() {
  local file=$1
  if [ -f "$file" ]; then
    sed -i.bak '/# >>> SHELL LOGGER >>>/,/# <<< SHELL LOGGER <<</d' "$file"
    sed -i.bak '/source .*shell-logger\/logger-snippets\/.*/d' "$file"
    echo "🧼 Cleaned $file"
  fi
}

cleanup_file "$HOME/.bashrc"
cleanup_file "$HOME/.zshrc"
cleanup_file "$HOME/.config/fish/config.fish"

rm -rf "$HOME/shell-logger"

echo "🧽 Shell logger fully removed. Restart your shell to revert."
