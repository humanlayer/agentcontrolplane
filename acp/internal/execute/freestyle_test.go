package execute

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ExampleFreestyleClient demonstrates how to use the Freestyle client
func TestExampleFreestyleClient(t *testing.T) {
	// Create a new client
	client := NewFreestyleClient(
		"Bj5Xq4Fa1R8DfdkwC36vHD-6QpekCWgHeJmfyR5JZ7cZJ5Paansm8VZ3km3B9AiLLK1",
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example script that calculates factorials
	script := `export default () => {
  // get the value of the factorials of the numbers from 1 to 10 combined
  const a = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];

  function factorial(n) {
    if (n === 0) {
      return 1;
    }
	throw new Error("test error");
    return n * factorial(n - 1);
  }

  const b = a.map(factorial);

  return b.reduce((a, b) => a + b);
};`

	// Create configuration
	config := FreestyleConfig{}

	// Execute the script
	resp, err := client.ExecuteScript(ctx, script, config)
	if err != nil {
		fmt.Printf("Error executing script: %v\n", err)
		return
	}

	// Print the result
	fmt.Printf("Script execution result: %v %v\n", resp.Result, resp.Error)
}
