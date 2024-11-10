#!/bin/sh
# POSIX-compliant script to log command details to a Unix socket

# Parameters:
# $1 - Execution phase (start/end)
# $2 - Command to log
# $3 - Working directory
# $4 - User who executed the command
# $5 - Unique identifier
# #6 - PID of the command
# $6 - Command result (success/failure)
# $7 - Exit status

# UNIX socket path
SOCKET_PATH="{{.SocketPath}}"

# Function to check command existence
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check if nc supports the -U flag
nc_supports_U() {
    error_message=$(echo | nc -U "" 2>&1)
    if echo "$error_message" | grep -q 'invalid option'; then
        return 1
    else
        return 0
    fi
}

# Python function to communicate via socket
send_via_python() {
  python -c "import socket; import sys; \
    s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM); \
    s.connect('$SOCKET_PATH'); \
    s.sendall('$LOG_MESSAGE'.encode('utf-8')); \
    s.close()" 2>/dev/null
}

# Perl function to communicate via socket
send_via_perl() {
  perl -e "use IO::Socket::UNIX; \
    \$sock = IO::Socket::UNIX->new(Type => SOCK_STREAM, Peer => '$SOCKET_PATH'); \
    print \$sock '$LOG_MESSAGE'; \
    close(\$sock);" 2>/dev/null
}

# Construct the log message including result and exit status
LOG_MESSAGE="$1|$2|$3|$4|$5|$6|$7|$8"

# Send the log message to the Go application via UNIX socket
if command_exists nc && nc_supports_U; then
  echo "$LOG_MESSAGE" | nc -U "$SOCKET_PATH"
elif command_exists socat; then
  echo "$LOG_MESSAGE" | socat - UNIX-CONNECT:"$SOCKET_PATH"
elif command_exists python; then
  send_via_python
elif command_exists perl; then
  send_via_perl
else
  # Output error if no suitable command is available
  echo "Neither nc, socat, python or perl are available on this system." >&2
  exit 1
fi
