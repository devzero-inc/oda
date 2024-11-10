generate_uuid() {
  echo "$(date +%s)-$$-$RANDOM"
}

generate_ppid() {
  echo "$$"
}

preexec() {
  export LAST_COMMAND=$1
  UUID=$(generate_uuid)
  PID=$(generate_ppid)
  # Send a start execution message
  {{.CommandScriptPath}} "start" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID"
}

precmd() {
  local exit_status=$?
  local result="success"
  
  if [[ $exit_status -ne 0 ]]; then
    result="failure"
  fi
  
  # Send an end execution message with result and exit status
  {{.CommandScriptPath}} "end" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID" "$result" "$exit_status"
}
