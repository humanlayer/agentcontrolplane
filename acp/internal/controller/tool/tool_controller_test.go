package tool

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
	"github.com/humanlayer/agentcontrolplane/acp/test/utils"
)

var _ = Describe("Tool Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-tool"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		AfterEach(func() {
			By("Cleanup the specific resource instance Tool")
			resource := &acp.Tool{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}
		})

		It("should successfully reconcile a function tool", func() {
			By("creating the custom resource")
			resource := &acp.Tool{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: acp.ToolSpec{
					ToolType:    "function",
					Name:        "add",
					Description: "Add two numbers",
					Parameters: runtime.RawExtension{
						Raw: []byte(`{
							"type": "object",
							"properties": {
								"a": {
									"type": "number"
								},
								"b": {
									"type": "number"
								}
							},
							"required": ["a", "b"]
						}`),
					},
					Execute: acp.ToolExecute{
						Builtin: &acp.BuiltinToolSpec{
							Name: "add",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			By("Reconciling the created resource")
			eventRecorder := record.NewFakeRecorder(10)
			controllerReconciler := &ToolReconciler{
				Client:   k8sClient,
				Scheme:   k8sClient.Scheme(),
				recorder: eventRecorder,
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the resource status")
			updatedTool := &acp.Tool{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTool)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedTool.Status.Ready).To(BeTrue())
			Expect(updatedTool.Status.Status).To(Equal("Ready"))
			Expect(updatedTool.Status.StatusDetail).To(Equal("Tool validation successful"))

			By("checking that a success event was created")
			utils.ExpectRecorder(eventRecorder).ToEmitEventContaining("ValidationSucceeded")
		})
	})
})
