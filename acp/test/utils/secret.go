package utils

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestSecret struct {
	Name      string
	Secret    *v1.Secret
	k8sClient client.Client
}

func (t *TestSecret) Setup(ctx context.Context, k8sClient client.Client) *v1.Secret {
	t.k8sClient = k8sClient

	By("creating the secret")
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.Name,
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("test-api-key"),
		},
	}
	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred())
	t.Secret = secret
	return secret
}

func (t *TestSecret) Teardown(ctx context.Context) {
	if t.k8sClient == nil {
		return
	}

	By("deleting the secret")
	Expect(t.k8sClient.Delete(ctx, t.Secret)).To(Succeed())
}
