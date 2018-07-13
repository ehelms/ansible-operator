package stub

import (
	"context"

	"github.com/automationbroker/ansible-operator/pkg/runner"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewHandler(m map[schema.GroupVersionKind]runner.Runner) sdk.Handler {
	return &Handler{crdToPlaybook: m}
}

type Handler struct {
	crdToPlaybook map[schema.GroupVersionKind]runner.Runner
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	p, ok := h.crdToPlaybook[event.Object.GetObjectKind().GroupVersionKind()]
	if !ok {
		logrus.Warnf("unable to find playbook mapping for gvk: %v", event.Object.GetObjectKind().GroupVersionKind())
		return nil
	}
	u, ok := event.Object.(*unstructured.Unstructured)
	if !ok {
		logrus.Warnf("object was not unstructured - %#v", event.Object)
		return nil
	}
	s := u.Object["spec"]
	spec, ok := s.(map[string]interface{})
	if !ok {
		u.Object["spec"] = map[string]interface{}{}
		sdk.Update(u)
		logrus.Warnf("spec is not a map[string]interface{} - %#v", s)
		return nil
	}

	je, err := p.Run(spec, u.GetName(), u.GetNamespace())
	if err != nil {
		return err
	}
	statusMap, ok := u.Object["status"].(map[string]interface{})
	if !ok {
		u.Object["status"] = runner.ResourceStatus{
			Status: runner.NewStatusFromStatusJobEvent(je),
		}
		sdk.Update(u)
		logrus.Infof("adding status for the first time")
		return nil
	}
	// Need to conver the map[string]interface into a resource status.
	rs := runner.UpdateResourceStatus(statusMap, je)
	u.Object["status"] = rs
	sdk.Update(u)
	return nil
}
