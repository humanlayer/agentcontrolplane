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

package llm

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

const (
	statusPending = "Pending"
	statusReady   = "Ready"
	statusError   = "Error"
)

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=llms/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	stateMachine *StateMachine
}

//
// llms.withTools can be used for passing in tools
// This is in options.go in langchaingo/llms/
// WithTools will add an option to set the tools to use.
// func WithTools(tools []Tool) CallOption {
// 	return func(o *CallOptions) {
// 		o.Tools = tools
// 	}
// }

// Some providers can have a base url. Here is an example of a base url	for OpenAI.
// This is in openaillm_option.go in langchaingo/llms/openai/
// WithBaseURL passes the OpenAI base url to the client. If not set, the base url
// is read from the OPENAI_BASE_URL environment variable. If still not set in ENV
// VAR OPENAI_BASE_URL, then the default value is https://api.openai.com/v1 is used.
//
//	func WithBaseURL(baseURL string) Option {
//		return func(opts *options) {
//			opts.baseURL = baseURL
//		}
//	}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var llm acp.LLM
	if err := r.Get(ctx, req.NamespacedName, &llm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure state machine is initialized
	if r.stateMachine == nil {
		r.stateMachine = NewStateMachine(r.Client, r.recorder)
	}

	// Delegate to StateMachine
	return r.stateMachine.Process(ctx, &llm)
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("llm-controller")
	r.stateMachine = NewStateMachine(r.Client, r.recorder)
	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.LLM{}).
		Named("llm").
		Complete(r)
}
