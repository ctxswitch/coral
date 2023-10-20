package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"
	builder "stvz.io/coral/pkg/controller/builder"
)

type ControllerOpts struct{}

type Controller struct{}

func SetupWithManager(mgr ctrl.Manager) (err error) {
	builderController := &builder.Controller{}
	if err = builderController.SetupWithManager(mgr); err != nil {
		return
	}

	// TODO: setup sync controller

	return
}
