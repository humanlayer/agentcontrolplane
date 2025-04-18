# /// script
# dependencies = [
#   "requests",
# ]
# ///

#!/usr/bin/env python3
import os
import random
import time
import json
import sys
import requests #  type: ignore
from requests.exceptions import ConnectionError, Timeout, RequestException

# Configuration for retry mechanism
MAX_RETRIES = 3
RETRY_DELAY = 2  # seconds between retries

# Current time in nanoseconds
current_time_ns = time.time_ns()

# Generate logs
logs_payload = {
    "resourceLogs": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "curl-test-service"}
                    }
                ]
            },
            "scopeLogs": [
                {
                    "scope": {
                        "name": "curl-test-scope"
                    },
                    "logRecords": [
                        {
                            "timeUnixNano": str(current_time_ns),
                            "severityNumber": 9,
                            "severityText": "INFO",
                            "body": {
                                "stringValue": "Hello from test service!"
                            },
                            "attributes": [
                                {
                                    "key": "service.name",
                                    "value": {
                                        "stringValue": "curl-test-service"
                                    }
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}

# Generate metrics
metrics_payload = {
    "resourceMetrics": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "curl-test-service"}
                    }
                ]
            },
            "scopeMetrics": [
                {
                    "scope": {
                        "name": "curl-test-scope"
                    },
                    "metrics": [
                        {
                            "name": "curl.test.metric",
                            "description": "Test metric",
                            "unit": "1",
                            "gauge": {
                                "dataPoints": [
                                    {
                                        "timeUnixNano": str(current_time_ns),
                                        "asInt": random.randint(0, 100)
                                    }
                                ]
                            }
                        }
                    ]
                }
            ]
        }
    ]
}

# Generate trace with parent-child relationship
trace_id_hex = os.urandom(16).hex()
parent_span_id_hex = os.urandom(8).hex()
child_span_id_hex = os.urandom(8).hex()

trace_payload = {
    "resourceSpans": [
        {
            "resource": {
                "attributes": [
                    {
                        "key": "service.name",
                        "value": {"stringValue": "python-random-service"}
                    }
                ]
            },
            "scopeSpans": [
                {
                    "scope": {
                        "name": "python-random-scope"
                    },
                    "spans": [
                        {
                            "traceId": trace_id_hex,
                            "spanId": parent_span_id_hex,
                            "name": "parent-operation",
                            "kind": "SPAN_KIND_SERVER",
                            "startTimeUnixNano": str(current_time_ns),
                            "endTimeUnixNano": str(current_time_ns + 30_000_000_000),  # 30 seconds later
                            "attributes": [
                                {
                                    "key": "operation.type",
                                    "value": {"stringValue": "parent"}
                                }
                            ]
                        },
                        {
                            "traceId": trace_id_hex,
                            "spanId": child_span_id_hex,
                            "parentSpanId": parent_span_id_hex,
                            "name": "child-operation",
                            "kind": "SPAN_KIND_INTERNAL",
                            "startTimeUnixNano": str(current_time_ns + 5_000_000_000),  # 5 seconds after parent starts
                            "endTimeUnixNano": str(current_time_ns + 15_000_000_000),  # 10 seconds duration
                            "attributes": [
                                {
                                    "key": "operation.type",
                                    "value": {"stringValue": "child"}
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}

url_base = "http://localhost:4318/v1"
headers = {"Content-Type": "application/json"}


def send_telemetry(endpoint, payload, telemetry_type):
    """
    Send telemetry data to the OpenTelemetry endpoint with retry logic
    
    Args:
        endpoint (str): The endpoint URL
        payload (dict): The telemetry data payload
        telemetry_type (str): Type of telemetry (logs, metrics, traces)
        
    Returns:
        tuple: (success (bool), response or error message (str))
    """
    url = f"{url_base}/{endpoint}"
    
    for attempt in range(MAX_RETRIES):
        try:
            print(f"Sending {telemetry_type} (attempt {attempt+1}/{MAX_RETRIES})...")
            response = requests.post(url, headers=headers, json=payload)
            response.raise_for_status()  # Raise exception for 4XX/5XX responses
            return True, response
        except ConnectionError as e:
            error_msg = f"Connection error: The OpenTelemetry endpoint at {url} is unavailable"
            print(f"{error_msg}: {str(e)}")
        except Timeout as e:
            error_msg = f"Timeout error: The request to {url} timed out"
            print(f"{error_msg}: {str(e)}")
        except RequestException as e:
            error_msg = f"Request error: Failed to send {telemetry_type} to {url}"
            print(f"{error_msg}: {str(e)}")
        
        # Don't sleep after the last attempt
        if attempt < MAX_RETRIES - 1:
            print(f"Retrying in {RETRY_DELAY} seconds...")
            time.sleep(RETRY_DELAY)
    
    return False, f"Failed to send {telemetry_type} after {MAX_RETRIES} attempts"

# Track overall success
success_count = 0
total_operations = 3

# Send logs
success, result = send_telemetry("logs", logs_payload, "logs")
if success:
    response = result
    print("Logs Status code:", response.status_code)
    print("Logs Response body:", response.text)
    success_count += 1
else:
    print(f"Logs Error: {result}")

# Send metrics
success, result = send_telemetry("metrics", metrics_payload, "metrics")
if success:
    response = result
    print("Metrics Status code:", response.status_code)
    print("Metrics Response body:", response.text)
    success_count += 1
else:
    print(f"Metrics Error: {result}")

# Send traces
success, result = send_telemetry("traces", trace_payload, "traces")
if success:
    response = result
    print("Traces Status code:", response.status_code)
    print("Traces Response body:", response.text)
    success_count += 1
else:
    print(f"Traces Error: {result}")

# Always print the generated IDs regardless of success
print("Generated traceId:", trace_id_hex)
print("Generated parent spanId:", parent_span_id_hex)
print("Generated child spanId:", child_span_id_hex)

# Summary
print(f"\nSummary: {success_count}/{total_operations} operations completed successfully")

# Exit with appropriate status code
if success_count == 0:
    print("All telemetry operations failed. Check if the OpenTelemetry endpoint is available.")
    sys.exit(1)
elif success_count < total_operations:
    print("Some telemetry operations failed. Check the logs for details.")
    sys.exit(1)
else:
    print("All telemetry operations completed successfully.")
    sys.exit(0)