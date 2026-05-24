#!/bin/sh
# e2e.sh — end-to-end and sandbox escape tests for xmd
# Requires the xmd binary to be built and on PATH or at $XMD_BIN
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
XMD="${XMD_BIN:-$SCRIPT_DIR/../bin/xmd}"
# Resolve to absolute path so cd in subtests doesn't break it
case "$XMD" in
  /*) ;;
  *)  XMD="$(cd "$(dirname "$XMD")" && pwd)/$(basename "$XMD")" ;;
esac

PASS=0
FAIL=0
SKIP=0

# Colors (if terminal supports them)
if [ -t 1 ]; then
  RED='\033[0;31m'
  GREEN='\033[0;32m'
  NC='\033[0m'
else
  RED=''
  GREEN=''
  NC=''
fi

pass() {
  PASS=$((PASS + 1))
  printf "${GREEN}PASS${NC}: %s\n" "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  printf "${RED}FAIL${NC}: %s\n" "$1"
  if [ -n "$2" ]; then
    echo "  $2"
  fi
}

skip() {
  SKIP=$((SKIP + 1))
  printf "SKIP: %s\n" "$1"
}

# ---- Inline test scripts ----
# These are self-contained; no dependency on an examples directory.
TEST_TMPDIR="$(mktemp -d /tmp/xmd-e2e-XXXXXX)"
HELLO_SCRIPT="$TEST_TMPDIR/hello.md"
SUMMARIZE_SCRIPT="$TEST_TMPDIR/summarize.md"

cat > "$HELLO_SCRIPT" << 'EOF'
---
xmd: 1
description: Greet someone by name
vars:
  name: required
---
Say hello to $NAME.
EOF

cat > "$SUMMARIZE_SCRIPT" << 'EOF'
---
xmd: 1
description: Summarize a file
vars:
  file: required
---
Summarize the file at $FILE.
EOF

# ---- Helper: mock backend ----
MOCK_PORT=11999
MOCK_PID=""

# Write the python server script to a temp file to avoid shell quoting issues.
# The server streams a proper SSE response so the completion backend's scanner
# picks up the content field correctly.
MOCK_SERVER_PY="$(mktemp /tmp/xmd-mock-XXXXXX.py)"
cat > "$MOCK_SERVER_PY" << 'PYEOF'
import http.server, sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 11999
SSE_RESPONSE = (
    'data: {"choices":[{"delta":{"content":"mock response"}}]}\n\n'
    'data: [DONE]\n\n'
).encode()
MODELS = b'{"data":[{"id":"mock-model"}]}'

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if '/models' in self.path:
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            self.wfile.write(MODELS)
        else:
            self.send_response(404)
            self.end_headers()
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        self.rfile.read(length)
        self.send_response(200)
        self.send_header('Content-Type', 'text/event-stream')
        self.end_headers()
        self.wfile.write(SSE_RESPONSE)
        self.wfile.flush()
    def log_message(self, *args): pass

class ReusableHTTPServer(http.server.HTTPServer):
    allow_reuse_address = True

ReusableHTTPServer(('127.0.0.1', PORT), Handler).serve_forever()
PYEOF

trap 'rm -rf "$TEST_TMPDIR" "$MOCK_SERVER_PY"; [ -n "$MOCK_PID" ] && { kill "$MOCK_PID" 2>/dev/null; wait "$MOCK_PID" 2>/dev/null; }; true' EXIT

start_mock_backend() {
  if [ -n "$MOCK_PID" ]; then
    kill "$MOCK_PID" 2>/dev/null || true
    sleep 0.3
  fi
  # Kill any stale process from a previous run on this port.
  lsof -ti tcp:"$MOCK_PORT" 2>/dev/null | xargs kill -9 2>/dev/null || true
  python3 "$MOCK_SERVER_PY" "$MOCK_PORT" &
  MOCK_PID=$!
  sleep 0.5
  export XMD_COMPLETION_URL="http://127.0.0.1:$MOCK_PORT/v1/chat/completions"
  export XMD_BACKEND="openai_completion"
}

# ---- Test: binary exists and is executable ----
if [ -x "$XMD" ]; then
  pass "binary exists and is executable"
else
  fail "binary not found at $XMD"
  exit 1
fi

# ---- Test: --help with no script prints usage ----
OUTPUT="$("$XMD" --help 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 0 ] && echo "$OUTPUT" | grep -q "Usage"; then
  pass "--help with no script prints usage"
else
  fail "--help with no script should print usage" "exit=$RC output=$OUTPUT"
fi

# ---- Test: --help flag ----
OUTPUT="$("$XMD" --no-sandbox --help "$HELLO_SCRIPT" 2>&1)" && RC=0 || RC=$?
if echo "$OUTPUT" | grep -q "name"; then
  pass "--help shows variable list"
else
  fail "--help output missing variable info" "exit=$RC output=$OUTPUT"
fi

# ---- Test: missing required variable ----
OUTPUT="$("$XMD" --no-sandbox "$HELLO_SCRIPT" 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 2 ] && echo "$OUTPUT" | grep -qi "required"; then
  pass "missing required variable gives exit 2"
else
  fail "missing required variable should give exit 2" "exit=$RC output=$OUTPUT"
fi

# ---- Test: undeclared variable ----
OUTPUT="$("$XMD" --no-sandbox "$HELLO_SCRIPT" name=world extra=oops 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 2 ] && echo "$OUTPUT" | grep -qi "undeclared"; then
  pass "undeclared variable gives exit 2"
else
  fail "undeclared variable should give exit 2" "exit=$RC output=$OUTPUT"
fi

# ---- Test: malformed script ----
TMPSCRIPT="$(mktemp /tmp/xmd-test-XXXXXX.md)"
cat > "$TMPSCRIPT" << 'EOF'
---
xmd: 1
unknown_bad_field: oops
---
Body.
EOF
OUTPUT="$("$XMD" "$TMPSCRIPT" 2>&1)" && RC=0 || RC=$?
rm -f "$TMPSCRIPT"
if [ "$RC" -eq 2 ] && echo "$OUTPUT" | grep -qi "unknown"; then
  pass "unknown frontmatter field gives exit 2"
else
  fail "unknown frontmatter field should give exit 2" "exit=$RC output=$OUTPUT"
fi

# ---- Test: script file not found ----
OUTPUT="$("$XMD" /nonexistent/script.md 2>&1)" && RC=0 || RC=$?
if [ "$RC" -eq 2 ] && echo "$OUTPUT" | grep -qi "not found"; then
  pass "missing script gives exit 2 with error message"
else
  fail "missing script should give exit 2" "exit=$RC output=$OUTPUT"
fi

# ---- Mock backend tests ----
start_mock_backend

# Test: basic script runs with mock backend
OUTPUT="$("$XMD" --no-sandbox "$HELLO_SCRIPT" name=world 2>/dev/null)" && RC=0 || RC=$?
if [ "$RC" -eq 0 ] && [ -n "$OUTPUT" ]; then
  pass "script runs and produces output"
else
  fail "script run failed" "exit=$RC output=$OUTPUT"
fi

# Test: variable preamble reaches the backend
TMPFILE="$(mktemp /tmp/xmd-test-XXXXXX.txt)"
echo "This is a test document about the xmd runtime." > "$TMPFILE"
OUTPUT="$("$XMD" --no-sandbox "$SUMMARIZE_SCRIPT" file="$TMPFILE" 2>/dev/null)" && RC=0 || RC=$?
rm -f "$TMPFILE"
if [ "$RC" -eq 0 ] && [ -n "$OUTPUT" ]; then
  pass "variable preamble works end to end"
else
  fail "summarize script failed" "exit=$RC output=$OUTPUT"
fi

# ---- Sandbox tests ----
if [ "$(uname)" = "Darwin" ] || [ "$(uname)" = "Linux" ]; then
  TMPWORK="$(mktemp -d /tmp/xmd-work-XXXXXX)"
  SIMPLE_SCRIPT="$TMPWORK/simple.md"
  cat > "$SIMPLE_SCRIPT" << 'EOF'
---
xmd: 1
description: Simple sandbox test
---
Output the word "sandboxed".
EOF

  # Test: binary runs under sandbox (with mock backend, sandboxed)
  OUTPUT="$(cd "$TMPWORK" && "$XMD" "$SIMPLE_SCRIPT" 2>/dev/null)" && RC=0 || RC=$?
  if [ "$RC" -eq 0 ]; then
    pass "binary runs under sandbox"
  else
    fail "sandbox run failed" "exit=$RC output=$OUTPUT"
  fi

  rm -rf "$TMPWORK"

  # Test: sandbox blocks writes to the home directory.
  # Uses agent_command backend — no HTTP mock needed. The "agent" is a touch
  # command targeting a file in $HOME. We verify the file was not created.
  # If sandbox_apply is denied by a parent sandbox (e.g. Claude Code) the
  # policy is not applied and the test is skipped rather than failed.
  SANDBOX_ESC_TMP="$(mktemp -d /tmp/xmd-sandesc-XXXXXX)"
  SANDBOX_ESC_SCRIPT="$SANDBOX_ESC_TMP/touch.md"
  cat > "$SANDBOX_ESC_SCRIPT" << 'EOF'
---
xmd: 1
description: Sandbox write escape test
---
Touch a file outside the working directory.
EOF
  SANDBOX_ESC_TARGET="$HOME/xmd-sandbox-escape-test-$$.txt"
  SANDBOX_ESC_STDERR="$(cd "$SANDBOX_ESC_TMP" && XMD_BACKEND=agent_command XMD_AGENT_CMD="touch $SANDBOX_ESC_TARGET" "$XMD" "$SANDBOX_ESC_SCRIPT" 2>&1 >/dev/null)" || true
  if echo "$SANDBOX_ESC_STDERR" | grep -qE "sandbox_apply|bubblewrap not found|bwrap:"; then
    skip "sandbox write escape test (sandbox policy could not be applied)"
  elif [ -f "$SANDBOX_ESC_TARGET" ]; then
    fail "sandbox did not block write to home directory"
    rm -f "$SANDBOX_ESC_TARGET"
  else
    pass "sandbox blocks writes to home directory"
  fi
  rm -rf "$SANDBOX_ESC_TMP"
fi

# Kill mock backend
kill "$MOCK_PID" 2>/dev/null || true; wait "$MOCK_PID" 2>/dev/null || true; MOCK_PID=""

# ---- Summary ----
echo ""
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
