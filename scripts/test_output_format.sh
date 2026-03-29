#!/bin/bash
# Test script for output format feature

BIN="./mcp_test"
FAILURES=0

echo "=== Output Format Tests ==="
echo ""

# Build binary
go build -o mcp_test .

# Test 1: JSON output (default)
echo "Test 1: JSON output (default)"
RESULT=$($BIN openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q '"success": true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 2: Explicit JSON output
echo "Test 2: Explicit JSON output"
RESULT=$($BIN --output json openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q '"success": true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 3: YAML output
echo "Test 3: YAML output"
RESULT=$($BIN --output yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q 'success: true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 4: Text output (compact JSON - no indentation)
echo "Test 4: Text output (compact JSON)"
RESULT=$($BIN --output text openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
# Text format should be compact JSON (no 2-space indentation)
if echo "$RESULT" | grep -q '"success":true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 5: Short flag -o
echo "Test 5: Short flag -o yaml"
RESULT=$($BIN -o yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q 'success: true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 6: YAML output has proper structure
echo "Test 6: YAML output has data and meta"
RESULT=$($BIN --output yaml openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q 'data:'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi
if echo "$RESULT" | grep -q 'meta:'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 7: JSON output structure
echo "Test 7: JSON output structure"
RESULT=$($BIN --output json openDeepWiki list_repositories --yaml 'limit: 1' 2>&1)
if echo "$RESULT" | grep -q '"success": true'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi
if echo "$RESULT" | grep -q '"data":'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi
if echo "$RESULT" | grep -q '"meta":'; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Test 8: Invalid output format should error
echo "Test 8: Invalid output format should error"
if $BIN --output invalid openDeepWiki list_repositories 2>&1 | grep -q "invalid output format"; then
  echo "  ✓ PASS"
else
  echo "  ✗ FAIL"
  FAILURES=$((FAILURES + 1))
fi

# Cleanup
rm -f mcp_test

echo ""
if [ $FAILURES -gt 0 ]; then
  echo "=== Tests completed with $FAILURES failure(s) ==="
  exit 1
else
  echo "=== All tests passed ==="
fi