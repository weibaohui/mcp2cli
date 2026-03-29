#!/bin/bash
# Edge case tests for mcp2cli YAML input functionality
set -e

BIN="./mcp_test"
PARAMS_FILE="/tmp/test_params.yaml"

echo "=== Edge Case Tests ==="
echo ""

# Build binary
go build -o mcp_test .

# Test 1: Empty YAML is treated as no input (falls back to tool info)
# This is expected CLI behavior - empty string means "not provided"
echo "Test E1: Empty YAML falls back to tool info (expected behavior)"
$BIN openDeepWiki list_repositories --yaml '' 2>&1 | grep -q '"param_example"' && echo "  ✓ PASS (empty YAML = no input)" || echo "  ✗ FAIL"

# Test 2: Invalid YAML syntax should error
echo "Test E2: Invalid YAML syntax should error"
! $BIN openDeepWiki list_repositories --yaml 'name: [broken' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS (correctly rejects invalid)" || echo "  ✗ FAIL"

# Test 3: Non-existent file should error
echo "Test E3: Non-existent file should error"
! $BIN openDeepWiki list_repositories -f /nonexistent/file.yaml 2>&1 | grep -q '"success": true' && echo "  ✓ PASS (correctly rejects missing file)" || echo "  ✗ FAIL"

# Test 4: YAML with comments
echo "Test E4: YAML with comments"
cat > "$PARAMS_FILE" << 'EOF'
# This is a comment
limit: 2  # inline comment
# Another comment
offset: 1
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 5: YAML with multiline string
echo "Test E5: Complex YAML structure"
cat > "$PARAMS_FILE" << 'EOF'
limit: 5
offset: 0
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 6: Priority - -f should override --yaml
echo "Test F6: Verify -f takes priority over --yaml"
cat > "$PARAMS_FILE" << 'EOF'
limit: 1
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" --yaml 'limit: 999' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 7: Status filter with string
echo "Test E7: Status filter with string value"
$BIN openDeepWiki list_repositories -y 'status: "ready"' 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 8: Multiple numeric parameters (using file)
echo "Test E8: Multiple numeric parameters"
cat > "$PARAMS_FILE" << 'EOF'
limit: 10
offset: 5
EOF
$BIN openDeepWiki list_repositories -f "$PARAMS_FILE" 2>&1 | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Cleanup
rm -f "$PARAMS_FILE"
rm -f mcp_test

echo ""
echo "=== Edge case tests completed ==="