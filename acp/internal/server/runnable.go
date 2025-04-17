package server

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RunnableServer implements manager.Runnable interface for starting and stopping the API server
type RunnableServer struct {
	server *APIServer
}

// NewRunnableServer creates a new server that can be added to the controller manager
func NewRunnableServer(client client.Client, port string) *RunnableServer {
	return &RunnableServer{
		server: NewAPIServer(client, port),
	}
}

// Start starts the API server and implements the manager.Runnable interface
func (r *RunnableServer) Start(ctx context.Context) error {
	return r.server.Start(ctx)
}

// NeedLeaderElection implements the LeaderElectionRunnable interface
func (r *RunnableServer) NeedLeaderElection() bool {
	// Only run on the leader if multiple instances are deployed
	return true
}

// AddToManager adds the API server to a controller manager
func AddToManager(mgr manager.Manager, port string) error {
	server := NewRunnableServer(mgr.GetClient(), port)
	return mgr.Add(server)
}
