package llmclient

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Schema", func() {
	It("represents a simple string type", func() {
		schema := &Schema{Type: "string"}
		jsonData, err := json.Marshal(schema)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(Equal(`{"type":"string"}`))
	})

	It("represents an array of strings", func() {
		schema := &Schema{
			Type:  "array",
			Items: &Schema{Type: "string"},
		}
		jsonData, err := json.Marshal(schema)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"array","items":{"type":"string"}}`))
	})

	It("represents a nested object with an array", func() {
		schema := &Schema{
			Type: "object",
			Properties: map[string]*Schema{
				"items": {
					Type:  "array",
					Items: &Schema{Type: "string"},
				},
			},
		}
		jsonData, err := json.Marshal(schema)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"object","properties":{"items":{"type":"array","items":{"type":"string"}}}}`))
	})

	It("includes enum values when provided", func() {
		schema := &Schema{
			Type: "string",
			Enum: []interface{}{"option1", "option2", "option3"},
		}
		jsonData, err := json.Marshal(schema)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"string","enum":["option1","option2","option3"]}`))
	})

	It("includes description when provided", func() {
		schema := &Schema{
			Type:        "boolean",
			Description: "A flag indicating whether the feature is enabled",
		}
		jsonData, err := json.Marshal(schema)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"boolean","description":"A flag indicating whether the feature is enabled"}`))
	})
})

var _ = Describe("ToolFunctionParameters", func() {
	It("represents a parameter with an array property", func() {
		params := ToolFunctionParameters{
			Type: "object",
			Properties: map[string]*Schema{
				"names": {
					Type:  "array",
					Items: &Schema{Type: "string"},
				},
			},
			Required: []string{"names"},
		}
		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())
		expected := `{"type":"object","properties":{"names":{"type":"array","items":{"type":"string"}}},"required":["names"]}`
		Expect(string(jsonData)).To(MatchJSON(expected))
	})

	It("represents a nested object parameter", func() {
		params := ToolFunctionParameters{
			Type: "object",
			Properties: map[string]*Schema{
				"data": {
					Type: "object",
					Properties: map[string]*Schema{
						"values": {
							Type:  "array",
							Items: &Schema{Type: "number"},
						},
					},
				},
			},
		}
		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())
		expected := `{"type":"object","properties":{"data":{"type":"object","properties":{"values":{"type":"array","items":{"type":"number"}}}}}}`
		Expect(string(jsonData)).To(MatchJSON(expected))
	})

	It("handles a complex schema with multiple properties and nested structures", func() {
		params := ToolFunctionParameters{
			Type: "object",
			Properties: map[string]*Schema{
				"name": {
					Type:        "string",
					Description: "The name of the item",
				},
				"tags": {
					Type:        "array",
					Description: "Tags associated with the item",
					Items:       &Schema{Type: "string"},
				},
				"metadata": {
					Type: "object",
					Properties: map[string]*Schema{
						"created": {Type: "string"},
						"size":    {Type: "number"},
						"features": {
							Type: "array",
							Items: &Schema{
								Type: "object",
								Properties: map[string]*Schema{
									"id":      {Type: "string"},
									"enabled": {Type: "boolean"},
								},
							},
						},
					},
				},
			},
			Required: []string{"name", "metadata"},
		}

		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())

		// Verify JSON structure after unmarshaling
		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		Expect(err).NotTo(HaveOccurred())

		Expect(result["type"]).To(Equal("object"))

		properties := result["properties"].(map[string]interface{})
		Expect(properties["name"].(map[string]interface{})["type"]).To(Equal("string"))
		Expect(properties["tags"].(map[string]interface{})["type"]).To(Equal("array"))

		metadata := properties["metadata"].(map[string]interface{})
		Expect(metadata["type"]).To(Equal("object"))

		metadataProps := metadata["properties"].(map[string]interface{})
		Expect(metadataProps["features"].(map[string]interface{})["type"]).To(Equal("array"))

		// Check deeply nested array of objects
		featuresItems := metadataProps["features"].(map[string]interface{})["items"].(map[string]interface{})
		Expect(featuresItems["type"]).To(Equal("object"))
		Expect(featuresItems["properties"].(map[string]interface{})["enabled"].(map[string]interface{})["type"]).To(Equal("boolean"))

		required := result["required"].([]interface{})
		Expect(required).To(ContainElement("name"))
		Expect(required).To(ContainElement("metadata"))
	})
})
