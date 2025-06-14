/*
Copyright 2025 the Agent Control Plane Authors.

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

package contactchannel

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	acp "github.com/humanlayer/agentcontrolplane/acp/api/v1alpha1"
)

var (
	// Status constants
	statusReady   = "Ready"
	statusError   = "Error"
	statusPending = "Pending"

	humanLayerAPIURL = "https://api.humanlayer.dev/humanlayer/v1/project"

	// Event reasons
	eventReasonInitializing        = "Initializing"
	eventReasonValidationFailed    = "ValidationFailed"
	eventReasonValidationSucceeded = "ValidationSucceeded"
)

// ContactChannelReconciler reconciles a ContactChannel object
type ContactChannelReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	recorder     record.EventRecorder
	stateMachine *StateMachine
}

// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=contactchannels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=contactchannels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=acp.humanlayer.dev,resources=contactchannels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile handles the reconciliation of ContactChannel resources
func (r *ContactChannelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var channel acp.ContactChannel
	if err := r.Get(ctx, req.NamespacedName, &channel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure state machine is initialized
	if r.stateMachine == nil {
		r.ensureStateMachine()
	}

	// Delegate to state machine
	return r.stateMachine.Process(ctx, &channel)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ContactChannelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("contactchannel-controller")

	// Initialize state machine
	r.stateMachine = NewStateMachine(r.Client, r.recorder)

	return ctrl.NewControllerManagedBy(mgr).
		For(&acp.ContactChannel{}).
		Named("contactchannel").
		Complete(r)
}

// ensureStateMachine initializes the state machine if not already initialized
func (r *ContactChannelReconciler) ensureStateMachine() {
	if r.stateMachine != nil {
		return
	}

	r.stateMachine = NewStateMachine(r.Client, r.recorder)
}
