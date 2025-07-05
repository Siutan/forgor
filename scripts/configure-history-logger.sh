#!/bin/bash
set -e

INSTALL_DIR="$HOME/shell-logger"
SNIPPET_DIR="$INSTALL_DIR/logger-snippets"
LOG_MARKER="# >>> SHELL LOGGER >>>"

mkdir -p "$SNIPPET_DIR"

# Create logger snippets with case-insensitive redaction and exit code logging
cat > "$SNIPPET_DIR/bash.sh" <<'EOF'
# >>> SHELL LOGGER >>>
export SHELL_SESSION_ID=${SHELL_SESSION_ID:-$(uuidgen)}
export SHELL_TTY=${SHELL_TTY:-$(tty)}

_forgor_sanitize_command() {
  local input="$1"
  echo "$input" | perl -pe 's/(token|key|secret|password|pwd|apikey)=\S+/$1=***/gi'
}

_forgor_log_command_bash() {
  local exit_code=$?
  local cmd_line
  # The command is the last one in the shell's history.
  local saved_opts
  saved_opts=$(set +o)
  set +H
  cmd_line=$(history 1)
  eval "$saved_opts"

  # History entry is like "  123  command". Strip the number.
  cmd_line=$(echo "$cmd_line" | sed 's/^[ ]*[0-9]\+[ ]*//')

  # Avoid logging empty commands or the logger itself.
  if [ -z "$cmd_line" ] || [[ "$cmd_line" == "_forgor_log_command_bash"* ]]; then
    return
  fi

  local sanitized_cmd_line=$(_forgor_sanitize_command "$cmd_line")
  echo "$EPOCHSECONDS|bash|$$|$SHELL_SESSION_ID|$SHELL_TTY|$PWD|$exit_code|$sanitized_cmd_line" >> ~/.command_log
}

# Append our logger to PROMPT_COMMAND to run before each prompt.
# Avoid adding it multiple times
if [[ -z "$PROMPT_COMMAND" || "$PROMPT_COMMAND" != *"_forgor_log_command_bash"* ]]; then
    PROMPT_COMMAND="_forgor_log_command_bash;${PROMPT_COMMAND}"
fi
# <<< SHELL LOGGER <<<
EOF

cat > "$SNIPPET_DIR/zsh.sh" <<'EOF'
# >>> SHELL LOGGER >>>
export SHELL_SESSION_ID=${SHELL_SESSION_ID:-$(uuidgen)}
export SHELL_TTY=${SHELL_TTY:-$(tty)}

_forgor_sanitize_command() {
  local input="$1"
  echo "$input" | perl -pe 's/(token|key|secret|password|pwd|apikey)=\S+/$1=***/gi'
}

# Using a global variable to pass the command from preexec to precmd
_forgor_cmd_to_log=""

_forgor_preexec_logger() {
  _forgor_cmd_to_log="$1"
}

_forgor_precmd_logger() {
  local exit_code=$?
  if [ -n "$_forgor_cmd_to_log" ]; then
    local cmd_line="$_forgor_cmd_to_log"
    _forgor_cmd_to_log="" # Unset after use

    local sanitized_cmd_line=$(_forgor_sanitize_command "$cmd_line")
    echo "$EPOCHSECONDS|zsh|$$|$SHELL_SESSION_ID|$SHELL_TTY|$PWD|$exit_code|$sanitized_cmd_line" >> ~/.command_log
  fi
}

# Ensure hooks are available and add our loggers
autoload -U add-zsh-hook
# Check if hooks are already there to avoid duplicates
if ! [[ "${preexec_functions[(r)_forgor_preexec_logger]}" == "_forgor_preexec_logger" ]]; then
    add-zsh-hook preexec _forgor_preexec_logger
fi
if ! [[ "${precmd_functions[(r)_forgor_precmd_logger]}" == "_forgor_precmd_logger" ]]; then
    add-zsh-hook precmd _forgor_precmd_logger
fi
# <<< SHELL LOGGER <<<
EOF

cat > "$SNIPPET_DIR/fish.fish" <<'EOF'
# >>> SHELL LOGGER >>>
if test -z "$SHELL_SESSION_ID"
    set -gx SHELL_SESSION_ID (uuidgen)
end
if test -z "$SHELL_TTY"
    set -gx SHELL_TTY (tty)
end

function _forgor_sanitize_command_fish
    set input $argv[1]
    echo $input | perl -pe 's/(token|key|secret|password|pwd|apikey)=\S+/$1=***/gi'
end

function _forgor_log_command_fish --on-event fish_postexec
    # $status contains the exit code of the last command.
    # $argv contains the command line of the last command.
    if test -z "$argv"
        return
    end

    set exit_code $status
    set full_cmd (string join " " $argv)
    set sanitized_cmd (_forgor_sanitize_command_fish "$full_cmd")
    echo (date +%s)"|fish|"(status pid)"|"$SHELL_SESSION_ID"|"$SHELL_TTY"|"$PWD"|"$exit_code"|"$sanitized_cmd" >> ~/.command_log
end
# <<< SHELL LOGGER <<<
EOF

# Inject into shell configs
inject_snippet() {
  local shell_name=$1
  local rc_file=$2
  local snippet_file=$3

  if ! grep -q "$LOG_MARKER" "$rc_file" 2>/dev/null; then
    echo -e "\n# Added by shell-logger\nsource \"$snippet_file\"" >> "$rc_file"
    echo "✅ Patched $rc_file for $shell_name"
  else
    echo "⚠️ $rc_file already contains logger hook"
  fi
}

detect_and_patch() {
  local shell=$(basename "$SHELL")
  case "$shell" in
    bash)
      inject_snippet "bash" "$HOME/.bashrc" "$SNIPPET_DIR/bash.sh"
      ;;
    zsh)
      inject_snippet "zsh" "$HOME/.zshrc" "$SNIPPET_DIR/zsh.sh"
      ;;
    fish)
      inject_snippet "fish" "$HOME/.config/fish/config.fish" "$SNIPPET_DIR/fish.fish"
      ;;
    *)
      echo "❌ Unsupported shell: $shell"
      exit 1
      ;;
  esac
}

detect_and_patch

echo "\n🚀 Shell logger installed with sensitive arg redaction. Restart your shell to activate."
