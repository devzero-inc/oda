function generate_uuid
    echo (date +%s)"-"(echo %self)"-"(random)
end

function generate_ppid
    echo (echo %self)
end

function fish_preexec --on-event fish_preexec
    set -gx LAST_COMMAND $argv[1]
    set -gx UUID (generate_uuid)
    set -gx PID (generate_ppid)
    # Send a start execution message
    {{.CommandScriptPath}} "start" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID"
end

function fish_postexec --on-event fish_postexec
    set -l exit_status $status
    set -l result "success"
    
    if test $exit_status -ne 0
        set result "failure"
    end
    
    # Send an end execution message with result and exit status
    {{.CommandScriptPath}} "end" "$LAST_COMMAND" "$PWD" "$USER" "$UUID" "$PID" "$result" "$exit_status"
end
