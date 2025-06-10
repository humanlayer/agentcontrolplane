package utils

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestMCPServer struct {
	Name                   string
	Transport              string
	Command                string
	Args                   []string
	URL                    string
	ApprovalContactChannel string
	MCPServer              *acp.MCPServer

	k8sClient client.Client
}

func (t *TestMCPServer) Setup(ctx context.Context, k8sClient client.Client) *acp.MCPServer {
	t.k8sClient = k8sClient

	By("creating the MCP server")
	mcpServer := &acp.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: acp.MCPServerSpec{
			Transport: t.Transport,
			Command:   t.Command,
			Args:      t.Args,
			URL:       t.URL,
		},
	}

	// Set default transport if not specified
	if mcpServer.Spec.Transport == "" {
		mcpServer.Spec.Transport = "stdio"
	}

	if t.ApprovalContactChannel != "" {
		mcpServer.Spec.ApprovalContactChannel = &acp.LocalObjectReference{
			Name: t.ApprovalContactChannel,
		}
	}

	_ = t.k8sClient.Delete(ctx, mcpServer) // Delete if exists
	err := t.k8sClient.Create(ctx, mcpServer)
	Expect(err).NotTo(HaveOccurred())
	Expect(t.k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, mcpServer)).To(Succeed())
	t.MCPServer = mcpServer
	return mcpServer
}

func (t *TestMCPServer) SetupWithStatus(
	ctx context.Context,
	k8sClient client.Client,
	status acp.MCPServerStatus,
) *acp.MCPServer {
	mcpServer := t.Setup(ctx, k8sClient)
	mcpServer.Status = status
	Expect(k8sClient.Status().Update(ctx, mcpServer)).To(Succeed())
	t.MCPServer = mcpServer
	return mcpServer
}

func (t *TestMCPServer) Teardown(ctx context.Context) {
	if t.k8sClient == nil || t.MCPServer == nil {
		return
	}
	By("deleting the MCP server")
	_ = t.k8sClient.Delete(ctx, t.MCPServer)
}
