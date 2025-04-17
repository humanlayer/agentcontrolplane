package utils

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.Name, Namespace: "default"}, secret)).To(Succeed())
	t.Secret = secret
	return secret
}

func (t *TestSecret) Teardown(ctx context.Context) {
	if t.k8sClient == nil || t.Secret == nil {
		return
	}

	By("deleting the secret")
	_ = t.k8sClient.Delete(ctx, t.Secret)
}
