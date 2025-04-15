package toolcall

import (
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
	. "github.com/onsi/ginkgo/v2"
	"go.opentelemetry.io/otel/trace/noop"
	"k8s.io/client-go/tools/record"
)

// reconciler creates a new reconciler for testing
func reconciler() (*ToolCallReconciler, *record.FakeRecorder) {
	By("creating a test reconciler")
	recorder := record.NewFakeRecorder(10)

	reconciler := &ToolCallReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
		Tracer:   noop.NewTracerProvider().Tracer("test"),
		MCPManager: &utils.MockMCPManager{
			NeedsApproval: false,
		},
	}

	return reconciler, recorder
}
