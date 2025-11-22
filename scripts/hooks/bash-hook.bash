# Added by Forgor - Bash integration

__forgor_preexec() {
  local cmd="$BASH_COMMAND"
  local timestamp=$(date +%s%N)
  export FORGOR_CURRENT_CMD_ID="${timestamp}-$$"
  export FORGOR_CMD_START_TIME="$timestamp"

  if [[ ! -S "$HOME/.config/forgor/daemon.sock" ]]; then
    return 0
  fi

  {
    echo "{\"type\":\"command_start\",\"payload\":{\"id\":\"$FORGOR_CURRENT_CMD_ID\",\"command\":\"${cmd//\"/\\\"}\",\"timestamp\":$timestamp,\"cwd\":\"$PWD\",\"shell\":\"bash\",\"shell_pid\":$$}}" \
    | nc -U -w 1 "$HOME/.config/forgor/daemon.sock" &>/dev/null &
  } &!
}

__forgor_precmd() {
  local exit_code=$?
  local end_time=$(date +%s%N)

  [[ -n "$FORGOR_CURRENT_CMD_ID" ]] || return 0
  [[ -S "$HOME/.config/forgor/daemon.sock" ]] || return 0

  local duration=$((end_time - FORGOR_CMD_START_TIME))

  {
    echo "{\"type\":\"command_end\",\"payload\":{\"id\":\"$FORGOR_CURRENT_CMD_ID\",\"exit_code\":$exit_code,\"duration_ns\":$duration}}" \
    | nc -U -w 1 "$HOME/.config/forgor/daemon.sock" &>/dev/null &
  } &!

  unset FORGOR_CURRENT_CMD_ID
  unset FORGOR_CMD_START_TIME
}

PROMPT_COMMAND="__forgor_precmd; $PROMPT_COMMAND"
trap '__forgor_preexec' DEBUG


