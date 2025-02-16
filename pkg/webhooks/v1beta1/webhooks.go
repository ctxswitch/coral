package v1beta1

import (
	"ctx.sh/coral/pkg/webhooks/v1beta1/imagesync"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func SetupWebhooksWithManager(mgr manager.Manager) error {
	w := imagesync.Webhook{}
	err := w.SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	return nil
}
