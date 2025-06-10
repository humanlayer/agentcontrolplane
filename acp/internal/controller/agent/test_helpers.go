package agent

import (
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewTestAgentReconciler creates an AgentReconciler for testing with simplified dependencies
func NewTestAgentReconciler(client client.Client, eventRecorder record.EventRecorder) *AgentReconciler {
	scheme := client.Scheme()

	return NewAgentReconciler(client, scheme, eventRecorder)
}
