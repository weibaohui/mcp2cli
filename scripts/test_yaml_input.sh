#!/bin/bash
# Test script for mcp2cli YAML input functionality
set -e

BIN="./mcp_test"
CONFIG_FILE="/tmp/mcp_test_config.json"
PARAMS_FILE="/tmp/test_params.yaml"

# Setup test config
mkdir -p ~/.config/mcp
cat > "$CONFIG_FILE" << 'EOF'
{
  "mcpServers": {
    "openDeepWiki": {
      "url": "https://opendeepwiki.k8m.site/mcp/streamable",
      "timeout": 30000
    }
  }
}
EOF

# Build binary
echo "Building mcp_test binary..."
go build -o mcp_test .

echo ""
echo "=== Test Suite: mcp2cli YAML Input ==="
echo ""

# Test 1: List servers (no args)
echo "Test 1: List servers (baseline)"
$BIN 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 2: List tools on server
echo "Test 2: List tools on server"
$BIN openDeepWiki 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 3: Get tool info (2 args, no YAML)
echo "Test 3: Get tool info (2 args, no YAML)"
$BIN openDeepWiki list_repositories 2>&1 | grep -q '"param_example"' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 4: Inline YAML with --yaml flag
echo "Test 4: Inline YAML with --yaml flag"
$BIN openDeepWiki list_repositories --yaml 'limit: 3' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 5: Inline YAML with -y flag
echo "Test 5: Inline YAML with -y flag"
$BIN openDeepWiki list_repositories -y 'limit: 2' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 6: Multi-line YAML
echo "Test 6: Multi-line YAML"
$BIN openDeepWiki list_repositories --yaml $'limit: 2\noffset: 1' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 7: YAML file input with -f
echo "Test 7: YAML file input with -f"
cat > "$PARAMS_FILE" << 'EOF'
limit: 3
offset: 0
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 8: YAML file with status filter
echo "Test 8: YAML file with status filter"
cat > "$PARAMS_FILE" << 'EOF'
limit: 2
status: "completed"
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 9: Stdin pipe input
echo "Test 9: Stdin pipe input"
echo 'limit: 2' | $BIN openDeepWiki list_repositories 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 10: Pipe YAML file to stdin
echo "Test 10: Pipe YAML file to stdin"
cat > "$PARAMS_FILE" << 'EOF'
limit: 1
EOF
cat "$PARAMS_FILE" | $BIN openDeepWiki list_repositories 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 11: Verify YAML parsing - check actual limit value
echo "Test 11: Verify YAML parsing - check actual limit value"
RESULT=$($BIN openDeepWiki list_repositories --yaml 'limit: 2' 2>&1)
# The result text field has escaped JSON, look for the unescaped limit
echo "$RESULT" | grep -q '\\"limit\\": 2' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 12: Backward compatibility - key=value format
echo "Test 12: Backward compatibility - key=value format"
$BIN openDeepWiki list_repositories limit=3 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 13: Backward compatibility - typed key:type=value
echo "Test 13: Backward compatibility - typed key:type=value"
$BIN openDeepWiki list_repositories limit:number=2 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 14: Help shows new flags
echo "Test 14: Help shows new YAML flags"
$BIN --help 2>&1 | grep -q '\-y, --yaml' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 15: Help shows file flag
echo "Test 15: Help shows file flag"
$BIN --help 2>&1 | grep -q '\-f, --file' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Cleanup
rm -f "$PARAMS_FILE"
rm -f mcp_test

echo ""
echo "=== All tests completed ==="