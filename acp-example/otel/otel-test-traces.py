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
import requests #  type: ignore

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

# Send logs
logs_response = requests.post(f"{url_base}/logs", headers=headers, json=logs_payload)
print("Logs Status code:", logs_response.status_code)
print("Logs Response body:", logs_response.text)

# Send metrics
metrics_response = requests.post(f"{url_base}/metrics", headers=headers, json=metrics_payload)
print("Metrics Status code:", metrics_response.status_code)
print("Metrics Response body:", metrics_response.text)

# Send traces
traces_response = requests.post(f"{url_base}/traces", headers=headers, json=trace_payload)
print("Traces Status code:", traces_response.status_code)
print("Traces Response body:", traces_response.text)
print("Generated traceId:", trace_id_hex)
print("Generated parent spanId:", parent_span_id_hex)
print("Generated child spanId:", child_span_id_hex)
