package adapters

import (
	testing "testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdapters(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Adapters Suite")
}
