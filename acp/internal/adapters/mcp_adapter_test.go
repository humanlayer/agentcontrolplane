package adapters

import (
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
	})
})
