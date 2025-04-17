package llmclient

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLLMClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Client Suite")
}
