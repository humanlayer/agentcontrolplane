package main

import (
	"context"
	"fmt"
	"log"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	// Create a context
	ctx := context.Background()

	// Get a k8s client
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Error getting k8s config: %v", err)
	}

	// Add our custom types to the scheme
	if err := acp.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("Error adding to scheme: %v", err)
	}

	// Create the client
	c, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Create a namespace for our example
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "agent-example",
		},
	}
	if err := c.Create(ctx, namespace); err != nil {
		log.Fatalf("Error creating namespace: %v", err)
	}
	defer func() {
		if err := c.Delete(ctx, namespace); err != nil {
			log.Printf("Error cleaning up namespace: %v", err)
		}
	}()

	// Create an MCP Server
	mcpServer := &acp.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-mcp",
			Namespace: namespace.Name,
		},
		Spec: acp.MCPServerSpec{
			Transport: "stdio",
			Command:   "python3",
			Args:      []string{"-m", "mcp_server"},
			Env: []acp.EnvVar{
				{
					Name:  "MODEL_NAME",
					Value: "gpt-4",
				},
			},
		},
	}

	if err := c.Create(ctx, mcpServer); err != nil {
		log.Fatalf("Error creating MCP server: %v", err)
	}

	// Create a Contact Channel
	contactChannel := &acp.ContactChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-contact",
			Namespace: namespace.Name,
		},
		Spec: acp.ContactChannelSpec{
			Email: &acp.EmailChannelConfig{
				Address: "example@example.com",
			},
		},
	}

	if err := c.Create(ctx, contactChannel); err != nil {
		log.Fatalf("Error creating contact channel: %v", err)
	}

	// Create an LLM
	llm := &acp.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-llm",
			Namespace: namespace.Name,
		},
		Spec: acp.LLMSpec{
			Provider: "openai",
			Parameters: acp.BaseConfig{
				Model: "gpt-4",
			},
			APIKeyFrom: &acp.APIKeySource{
				SecretKeyRef: acp.SecretKeyRef{
					Name: "openai-secret",
					Key:  "api-key",
				},
			},
		},
	}

	if err := c.Create(ctx, llm); err != nil {
		log.Fatalf("Error creating LLM: %v", err)
	}

	// Create an Agent
	agent := &acp.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-agent",
			Namespace: namespace.Name,
		},
		Spec: acp.AgentSpec{
			System: "You are a helpful assistant that can answer questions.",
			LLMRef: acp.LocalObjectReference{
				Name: llm.Name,
			},
			MCPServers: []acp.LocalObjectReference{
				{Name: mcpServer.Name},
			},
			HumanContactChannels: []acp.LocalObjectReference{
				{Name: contactChannel.Name},
			},
		},
	}

	if err := c.Create(ctx, agent); err != nil {
		log.Fatalf("Error creating agent: %v", err)
	}

	// Create a Task
	task := &acp.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-task",
			Namespace: namespace.Name,
		},
		Spec: acp.TaskSpec{
			AgentRef: acp.LocalObjectReference{
				Name: agent.Name,
			},
			UserMessage: "What is the capital of France?",
		},
	}

	if err := c.Create(ctx, task); err != nil {
		log.Fatalf("Error creating task: %v", err)
	}

	fmt.Println("Successfully created example resources!")
	fmt.Println("You can check their status with:")
	fmt.Printf("kubectl get mcpserver -n %s\n", namespace.Name)
	fmt.Printf("kubectl get contactchannel -n %s\n", namespace.Name)
	fmt.Printf("kubectl get llm -n %s\n", namespace.Name)
	fmt.Printf("kubectl get agent -n %s\n", namespace.Name)
	fmt.Printf("kubectl get task -n %s\n", namespace.Name)
}
