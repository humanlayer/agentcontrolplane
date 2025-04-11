/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package toolcall

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

var (
	ctx       context.Context
	cancel    context.CancelFunc
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
		UseExistingCluster:    nil,
	}

	var err error
	// Start the test environment
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = apiextensionsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = acp.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Wait for CRDs to be ready
	timeout := time.Second * 60 // Increased timeout
	interval := time.Second * 2 // Increased interval

	By("waiting for CRDs to be ready")
	Eventually(func() error {
		crds := []string{
			"llms.acp.humanlayer.dev",
			"tools.acp.humanlayer.dev",
			"agents.acp.humanlayer.dev",
			"tasks.acp.humanlayer.dev",
			"toolcalls.acp.humanlayer.dev",
			"mcpservers.acp.humanlayer.dev",
			"contactchannels.acp.humanlayer.dev",
		}

		for _, crdName := range crds {
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
				return err
			}

			established := false
			for _, condition := range crd.Status.Conditions {
				if condition.Type == apiextensionsv1.Established &&
					condition.Status == apiextensionsv1.ConditionTrue {
					established = true
					break
				}
			}
			if !established {
				return fmt.Errorf("CRD %s not established", crdName)
			}
		}
		return nil
	}, timeout, interval).Should(Succeed())

	// Additional verification for specific CRDs
	Eventually(func() error {
		// Try to list all required CRD types
		lists := []client.ObjectList{
			&acp.LLMList{},
			&acp.ToolList{},
			&acp.AgentList{},
			&acp.TaskList{},
			&acp.ToolCallList{},
			&acp.MCPServerList{},
			&acp.ContactChannelList{},
		}

		for _, list := range lists {
			if err := k8sClient.List(ctx, list); err != nil {
				return err
			}
		}
		return nil
	}, timeout, interval).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
