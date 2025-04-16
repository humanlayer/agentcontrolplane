package llmclient

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tool Function Parameters", func() {
	It("represents a simple parameter", func() {
		params := ToolFunctionParameters{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}
		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"object","properties":{"name":{"type":"string"}}}`))
	})

	It("represents an array parameter", func() {
		params := ToolFunctionParameters{
			"type": "object",
			"properties": map[string]interface{}{
				"items": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}
		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(jsonData)).To(MatchJSON(`{"type":"object","properties":{"items":{"type":"array","items":{"type":"string"}}}}`))
	})

	It("represents a complex schema with nested objects and arrays", func() {
		params := ToolFunctionParameters{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the item",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Tags associated with the item",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"metadata": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"created": map[string]interface{}{
							"type": "string",
						},
						"size": map[string]interface{}{
							"type": "number",
						},
						"features": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"id": map[string]interface{}{
										"type": "string",
									},
									"enabled": map[string]interface{}{
										"type": "boolean",
									},
								},
							},
						},
					},
				},
			},
			"required": []string{"name", "metadata"},
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

	It("handles complex JSON Schema constructs like anyOf", func() {
		params := ToolFunctionParameters{
			"type": "object",
			"properties": map[string]interface{}{
				"options": map[string]interface{}{
					"anyOf": []interface{}{
						map[string]interface{}{
							"type": "string",
							"enum": []string{"option1", "option2"},
						},
						map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"customOption": map[string]interface{}{
									"type": "string",
								},
							},
							"required": []string{"customOption"},
						},
					},
				},
			},
		}

		jsonData, err := json.Marshal(params)
		Expect(err).NotTo(HaveOccurred())

		var result map[string]interface{}
		err = json.Unmarshal(jsonData, &result)
		Expect(err).NotTo(HaveOccurred())

		properties := result["properties"].(map[string]interface{})
		options := properties["options"].(map[string]interface{})
		Expect(options).To(HaveKey("anyOf"))

		anyOf := options["anyOf"].([]interface{})
		Expect(anyOf).To(HaveLen(2))

		firstOption := anyOf[0].(map[string]interface{})
		Expect(firstOption["type"]).To(Equal("string"))
		Expect(firstOption["enum"]).To(ContainElements("option1", "option2"))

		secondOption := anyOf[1].(map[string]interface{})
		Expect(secondOption["type"]).To(Equal("object"))
		Expect(secondOption).To(HaveKey("properties"))
		Expect(secondOption["required"]).To(ContainElement("customOption"))
	})
})
