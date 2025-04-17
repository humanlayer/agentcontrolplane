package utils

import (
	"context"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestContactChannel represents a test ContactChannel resource
type TestContactChannel struct {
	Name           string
	ChannelType    acp.ContactChannelType
	SecretName     string
	ContactChannel *acp.ContactChannel

	k8sClient client.Client
}

func (t *TestContactChannel) Setup(ctx context.Context, k8sClient client.Client) *acp.ContactChannel {
	t.k8sClient = k8sClient

	By("creating the contact channel")
	contactChannel := &acp.ContactChannel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Spec: acp.ContactChannelSpec{
			Type: t.ChannelType,
			APIKeyFrom: acp.APIKeySource{
				SecretKeyRef: acp.SecretKeyRef{
					Name: t.SecretName,
					Key:  "api-key",
				},
			},
		},
	}

	// Add specific config based on channel type
	if t.ChannelType == acp.ContactChannelTypeSlack {
		contactChannel.Spec.Slack = &acp.SlackChannelConfig{
			ChannelOrUserID: "C12345678",
		}
	} else if t.ChannelType == acp.ContactChannelTypeEmail {
		contactChannel.Spec.Email = &acp.EmailChannelConfig{
			Address: "test@example.com",
		}
	}

	_ = k8sClient.Delete(ctx, contactChannel) // Delete if exists
	err := k8sClient.Create(ctx, contactChannel)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, contactChannel)).To(Succeed())
	t.ContactChannel = contactChannel
	return contactChannel
}

func (t *TestContactChannel) SetupWithStatus(
	ctx context.Context,
	k8sClient client.Client,
	status acp.ContactChannelStatus,
) *acp.ContactChannel {
	contactChannel := t.Setup(ctx, k8sClient)
	contactChannel.Status = status
	Expect(k8sClient.Status().Update(ctx, contactChannel)).To(Succeed())
	t.ContactChannel = contactChannel
	return contactChannel
}

func (t *TestContactChannel) Teardown(ctx context.Context) {
	if t.k8sClient == nil || t.ContactChannel == nil {
		return
	}

	By("deleting the contact channel")
	_ = t.k8sClient.Delete(ctx, t.ContactChannel)
}
