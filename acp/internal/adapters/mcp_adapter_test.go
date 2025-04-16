package adapters

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

var _ = Describe("MCP Adapter", func() {
	Context("When converting MCP tools to LLM client tools", func() {
		It("should correctly format tool names with server prefix", func() {
			// Create sample tools with different names
			tools := []acp.MCPTool{
				{
					Name:        "fetch",
					Description: "Fetches data from URL",
					InputSchema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)},
				},
				{
					Name:        "calculate",
					Description: "Performs calculations",
					InputSchema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)},
				},
			}

			// Test with a server name different from tool names
			By("using a server name different from tool names")
			serverName := "server-alpha"
			result := ConvertMCPToolsToLLMClientTools(tools, serverName)

			// Verify correct naming
			Expect(result).To(HaveLen(2))
			Expect(result[0].Function.Name).To(Equal("server-alpha__fetch"))
			Expect(result[1].Function.Name).To(Equal("server-alpha__calculate"))

			// Test with a server name similar to a tool name (the bug scenario)
			By("using a server name that contains a tool name")
			serverName = "fetch-server"
			result = ConvertMCPToolsToLLMClientTools(tools, serverName)

			// Verify correct naming - even when server name contains tool name
			Expect(result).To(HaveLen(2))
			Expect(result[0].Function.Name).To(Equal("fetch-server__fetch"))
			Expect(result[1].Function.Name).To(Equal("fetch-server__calculate"))

			// Verify the bug case specifically - tool name should never be used as server name
			By("ensuring tool name is never used as server name (the bug case)")
			for _, tool := range result {
				Expect(tool.Function.Name).NotTo(Equal("fetch__fetch"))
				Expect(tool.Function.Name).NotTo(Equal("calculate__calculate"))
			}
		})

		It("should handle empty tool list", func() {
			serverName := "test-server"
			result := ConvertMCPToolsToLLMClientTools([]acp.MCPTool{}, serverName)
			Expect(result).To(BeEmpty())
		})

		It("converts a tool with an array parameter", func() {
			mcpTool := acp.MCPTool{
				Name:        "process_list",
				Description: "Processes a list of items",
				InputSchema: runtime.RawExtension{Raw: []byte(`{"type":"object","properties":{"items":{"type":"array","items":{"type":"string"}}}}`)},
			}
			clientTools := ConvertMCPToolsToLLMClientTools([]acp.MCPTool{mcpTool}, "test-server")

			Expect(clientTools).To(HaveLen(1))
			tool := clientTools[0]
			Expect(tool.Function.Name).To(Equal("test-server__process_list"))
			Expect(tool.Function.Parameters.Type).To(Equal("object"))
			Expect(tool.Function.Parameters.Properties).To(HaveKey("items"))
			itemsSchema := tool.Function.Parameters.Properties["items"]
			Expect(itemsSchema.Type).To(Equal("array"))
			Expect(itemsSchema.Items).NotTo(BeNil())
			Expect(itemsSchema.Items.Type).To(Equal("string"))
		})

		It("converts a tool with a complex nested schema", func() {
			complexSchema := `{
				"type": "object",
				"properties": {
					"names": {
						"type": "array",
						"items": {
							"type": "string"
						}
					},
					"options": {
						"type": "object",
						"properties": {
							"flag": {
								"type": "boolean"
							}
						}
					}
				},
				"required": ["names"]
			}`

			mcpTool := acp.MCPTool{
				Name:        "complex-tool",
				Description: "A tool with complex schema including arrays",
				InputSchema: runtime.RawExtension{Raw: []byte(complexSchema)},
			}

			clientTools := ConvertMCPToolsToLLMClientTools([]acp.MCPTool{mcpTool}, "test-server")

			Expect(clientTools).To(HaveLen(1))
			tool := clientTools[0]
			Expect(tool.Function.Parameters.Type).To(Equal("object"))
			Expect(tool.Function.Parameters.Required).To(ContainElement("names"))

			// Verify array parameter
			namesSchema := tool.Function.Parameters.Properties["names"]
			Expect(namesSchema.Type).To(Equal("array"))
			Expect(namesSchema.Items).NotTo(BeNil())
			Expect(namesSchema.Items.Type).To(Equal("string"))

			// Verify nested object parameter
			optionsSchema := tool.Function.Parameters.Properties["options"]
			Expect(optionsSchema.Type).To(Equal("object"))
			Expect(optionsSchema.Properties).NotTo(BeNil())
			Expect(optionsSchema.Properties["flag"].Type).To(Equal("boolean"))

			// Verify JSON serialization roundtrip
			jsonBytes, err := json.Marshal(tool.Function.Parameters)
			Expect(err).NotTo(HaveOccurred())

			var unmarshalled map[string]interface{}
			err = json.Unmarshal(jsonBytes, &unmarshalled)
			Expect(err).NotTo(HaveOccurred())

			properties := unmarshalled["properties"].(map[string]interface{})
			Expect(properties).To(HaveKey("names"))
			Expect(properties).To(HaveKey("options"))
		})
	})
})
