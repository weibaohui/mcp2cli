#!/bin/bash
# Test script for output format feature
set -e

BIN="./mcp_test"

echo "=== Output Format Tests ==="
echo ""

# Build binary
go build -o mcp_test .

# Test 1: JSON output (default)
echo "Test 1: JSON output (default)"
RESULT=$($BIN openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 2: Explicit JSON output
echo "Test 2: Explicit JSON output"
RESULT=$($BIN --output json openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 3: YAML output
echo "Test 3: YAML output"
RESULT=$($BIN --output yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q 'success: true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 4: Text output
echo "Test 4: Text output"
RESULT=$($BIN --output text openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 5: Short flag -o
echo "Test 5: Short flag -o yaml"
RESULT=$($BIN -o yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q 'success: true' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 6: YAML output has proper structure
echo "Test 6: YAML output has data and meta"
RESULT=$($BIN --output yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q 'data:' && echo "  ✓ PASS" || echo "  ✗ FAIL"
echo "$RESULT" | grep -q 'meta:' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Test 7: JSON output structure
echo "Test 7: JSON output structure"
RESULT=$($BIN --output json openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
echo "$RESULT" | grep -q '"success": true' && echo "  ✓ PASS" || echo "  ✗ FAIL"
echo "$RESULT" | grep -q '"data":' && echo "  ✓ PASS" || echo "  ✗ FAIL"
echo "$RESULT" | grep -q '"meta":' && echo "  ✓ PASS" || echo "  ✗ FAIL"

# Cleanup
rm -f mcp_test

echo ""
echo "=== Tests completed ==="