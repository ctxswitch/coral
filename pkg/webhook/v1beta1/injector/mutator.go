package injector

import (
	"encoding/json"
	"net/http"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Action struct {
	MutatePullPolicy bool
	MutateSelectors  bool
}

type Mutator struct {
	enabled  bool
	injected bool
	kind     string
	obj      client.Object
}

func FromReq(req admission.Request, decoder admission.Decoder) (*Mutator, error) {
	m := &Mutator{}

	m.kind = req.Kind.Kind

	obj, err := ObjectFromKind(m.kind)
	if err != nil {
		return nil, err
	}

	err = decoder.Decode(req, obj)
	if err != nil {
		return nil, err
	}
	m.obj = obj

	annotations := obj.GetAnnotations()
	enabled, ok := annotations[coralv1beta1.ImageSyncEnableAnnotation]
	if ok && enabled == "true" {
		m.enabled = true
	} else {
		// If the annotation is not set or is not "true", we do not manage this object.
		return m, nil
	}

	injected, ok := annotations[coralv1beta1.ImageSyncInjectedAnnotation]
	if ok && injected == "true" {
		m.injected = true
	}

	return m, nil
}

func (m *Mutator) Managed() bool {
	return m.enabled
}

func (m *Mutator) Mutate(req admission.Request) admission.Response {
	obj := m.mutate(m.obj)

	o, err := json.Marshal(obj)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, o)
}

func (m *Mutator) mutate(obj client.Object) client.Object {
	return obj
}
