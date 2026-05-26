#!/bin/bash
# Load testing script for one-time-link API

set -e

# Default parameters
CONCURRENT=10
REQUESTS=100
ENDPOINT="create"
BASE_URL="http://localhost:8080"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --concurrent)
            CONCURRENT="$2"
            shift 2
            ;;
        --requests)
            REQUESTS="$2"
            shift 2
            ;;
        --endpoint)
            ENDPOINT="$2"
            shift 2
            ;;
        --base-url)
            BASE_URL="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== Load Testing ==="
echo "Endpoint: $ENDPOINT"
echo "Concurrent: $CONCURRENT"
echo "Total Requests: $REQUESTS"
echo ""

# Test data
TEST_DATA='{"ciphertext":"dGVzdCBjaXBoZXJ0ZXh0IGZvciBsb2FkIHRlc3Rpbmc","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}'

# Determine endpoint URL
case $ENDPOINT in
    create)
        URL="$BASE_URL/api/secrets"
        METHOD="POST"
        ;;
    health)
        URL="$BASE_URL/healthz"
        METHOD="GET"
        ;;
    *)
        URL="$BASE_URL/api/secrets"
        METHOD="POST"
        ;;
esac

# Check if Apache Bench is available
if ! command -v ab &> /dev/null; then
    echo "Error: Apache Bench (ab) is not installed"
    echo "Install it with: apt-get install apache2-utils (Ubuntu/Debian)"
    echo "                 yum install httpd-tools (CentOS/RHEL)"
    echo "                 brew install apache2 (macOS)"
    exit 1
fi

echo "Starting load test..."
START_TIME=$(date +%s)

# Create temp file for POST data
if [ "$METHOD" = "POST" ]; then
    TEMP_FILE=$(mktemp)
    echo "$TEST_DATA" > "$TEMP_FILE"
    
    # Run Apache Bench
    ab -n "$REQUESTS" -c "$CONCURRENT" \
        -T "application/json" \
        -p "$TEMP_FILE" \
        "$URL" > /tmp/ab_results.txt 2>&1
    
    rm "$TEMP_FILE"
else
    # Run Apache Bench for GET
    ab -n "$REQUESTS" -c "$CONCURRENT" \
        "$URL" > /tmp/ab_results.txt 2>&1
fi

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "=== Results ==="

# Parse Apache Bench results
grep "Complete requests:" /tmp/ab_results.txt
grep "Failed requests:" /tmp/ab_results.txt
grep "Requests per second:" /tmp/ab_results.txt
grep "Time per request:" /tmp/ab_results.txt | head -1
echo ""

echo "Response Times (ms):"
grep "min)" /tmp/ab_results.txt | awk '{print "  Min: " $3 " Max: " $5 " Mean: " $7}'
grep "50%" /tmp/ab_results.txt | awk '{print "  P50: " $2}'
grep "95%" /tmp/ab_results.txt | awk '{print "  P95: " $2}'
grep "99%" /tmp/ab_results.txt | awk '{print "  P99: " $2}'
echo ""

# Performance assessment
P95=$(grep "95%" /tmp/ab_results.txt | awk '{print $2}')
FAILED=$(grep "Failed requests:" /tmp/ab_results.txt | awk '{print $3}')

if [ "$P95" -lt 50 ]; then
    echo "✓ Performance: Excellent (P95 < 50ms)"
elif [ "$P95" -lt 100 ]; then
    echo "✓ Performance: Good (P95 < 100ms)"
elif [ "$P95" -lt 200 ]; then
    echo "⚠ Performance: Acceptable (P95 < 200ms)"
else
    echo "✗ Performance: Poor (P95 >= 200ms)"
fi

if [ "$FAILED" -eq 0 ]; then
    echo "✓ Reliability: 100% success rate"
else
    SUCCESS_RATE=$(awk "BEGIN {print (($REQUESTS - $FAILED) / $REQUESTS) * 100}")
    if (( $(echo "$SUCCESS_RATE >= 99" | bc -l) )); then
        echo "✓ Reliability: ${SUCCESS_RATE}% success rate"
    elif (( $(echo "$SUCCESS_RATE >= 95" | bc -l) )); then
        echo "⚠ Reliability: ${SUCCESS_RATE}% success rate"
    else
        echo "✗ Reliability: ${SUCCESS_RATE}% success rate"
    fi
fi

echo ""
echo "=== Load Test Complete ==="

# Clean up
rm -f /tmp/ab_results.txt
