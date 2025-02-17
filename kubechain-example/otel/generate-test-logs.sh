#!/bin/bash
TIMESTAMP=$(date -u +%s%N)  # Get current UTC time in nanoseconds

cat > otel-test-logs.json << EOF
{
  "resourceLogs": [
    {
      "resource": {
        "attributes": [
          {
            "key": "service.name",
            "value": {
              "stringValue": "curl-test-service"
            }
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
              "timeUnixNano": "$TIMESTAMP",
              "severityNumber": 9,
              "severityText": "INFO",
              "body": {
                "stringValue": "Hello from curl!"
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
EOF
