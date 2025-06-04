package coral

import (
	"ctx.sh/coral/pkg/gen/coral/v1beta1/coralv1beta1connect"
	"ctx.sh/coral/pkg/store"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:skip
type Options struct {
	NodeRef *store.NodeRef
}

// +kubebuilder:skip
type Service struct {
	nodeRef *store.NodeRef
}

func SetupWebhookWithManager(mgr ctrl.Manager, opts *Options) error {
	service := &Service{
		nodeRef: opts.NodeRef,
	}

	path, handler := coralv1beta1connect.NewCoralServiceHandler(service)

	mgr.GetWebhookServer().Register(path, handler)

	return nil
}
