generate_uuid() {
    echo "$(date +%s)-$$-$RANDOM"
}

generate_ppid() {
  echo "$$"
}

preexec_invoke_exec() {
    # Avoid running preexec_invoke_exec for PROMPT_COMMAND
    if [[ "$BASH_COMMAND" != "$PROMPT_COMMAND" ]]; then
        export UUID=$(generate_uuid)
        export PID=$(generate_ppid)
        export LAST_COMMAND="$BASH_COMMAND"
        # Send a start execution message
        {{.CommandScriptPath}} "start" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID"
    fi
}
trap 'preexec_invoke_exec' DEBUG

precmd_invoke_cmd() {
    local exit_status=$?
    local result="success"

    if [[ $exit_status -ne 0 ]]; then
        result="failure"
    fi

    # Send an end execution message with the result and exit status
    {{.CommandScriptPath}} "end" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID" "$result" "$exit_status"
}

# Update PROMPT_COMMAND to invoke precmd_invoke_cmd
# Append precmd_invoke_cmd to PROMPT_COMMAND to run after each command
PROMPT_COMMAND="${PROMPT_COMMAND:+$PROMPT_COMMAND; }precmd_invoke_cmd"
