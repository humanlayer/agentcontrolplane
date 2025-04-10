package humanlayer

import (
	"context"
	"fmt"
	"log"
	"os"
)

func RunTest() {
	// Create a new client with your API key
	client := NewClient(os.Getenv("HUMANLAYER_API_KEY"))

	// Create a context
	ctx := context.Background()

	// Example function call
	response, err := client.Call(
		ctx,
		"run-123",  // Run ID
		"call-456", // Call ID
		map[string]interface{}{
			"fn": "example_function", // Function name
			"kwargs": map[string]interface{}{ // Arguments
				"param1": "value1",
				"param2": 42,
			},
		},
	)
	if err != nil {
		log.Fatalf("Error calling function: %v", err)
	}

	fmt.Printf("Initial response: %+v\n", response)

	// Poll for approval
	var finalResponse interface{}
	finalResponse, err = nil, fmt.Errorf("not implemented")
	if err != nil {
		log.Fatalf("Error polling for approval: %v", err)
	}

	fmt.Printf("Final response: %+v\n", finalResponse)
}
