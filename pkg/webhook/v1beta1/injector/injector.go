package injector

import (
	"context"
	"net/http"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Go imports

type Injector struct {
	client.Client
	cache   cache.Cache
	decoder admission.Decoder
	log     logr.Logger

	// default webhook action as config value
	defaultAction admission.Response
}

// SetupWebhookWithManager adds webhook for the resource injector.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	i := &Injector{
		Client:        mgr.GetClient(),
		cache:         mgr.GetCache(),
		decoder:       admission.NewDecoder(mgr.GetScheme()),
		defaultAction: admission.Allowed(""),
		log:           mgr.GetLogger().WithName("image-injector"),
	}

	mgr.GetWebhookServer().Register("/inject-coral-ctx-sh-v1beta1-imagesync", &webhook.Admission{
		Handler: i,
	})

	return nil
}

func (i *Injector) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := ctrl.LoggerFrom(ctx)
	log.V(6).Info("handling request", "req", req)

	mutator, err := FromReq(req, i.decoder)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// If we are not managing the object, then we should just allow it through
	if !mutator.Managed() {
		return admission.Allowed("")
	}

	return mutator.Mutate(req)
}

var _ admission.Handler = &Injector{}
