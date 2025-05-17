package image

import (
	"context"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type NodeOptions struct {
	Client client.Client
	Name   string
}

type Node struct {
	client client.Client
	name   string
	node   *corev1.Node
	sync.Mutex
}

func NewNode(opts *NodeOptions) *Node {
	return &Node{
		name:   opts.Name,
		client: opts.Client,
	}
}

func (n *Node) Update(ctx context.Context, img string) error {
	n.Lock()
	defer n.Unlock()

	log := ctrl.LoggerFrom(ctx, "node", n.name, "action", "update")

	if err := n.observe(ctx); err != nil {
		return err
	}

	fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
	ref := util.GetImageLabelValue(fqn)
	log.V(4).Info("updating node label", "ref", ref)

	n.addNodeLabel(coralv1beta1.ImageSyncLabel+"/"+ref, "present")

	return n.update(ctx)
}

func (n *Node) Remove(ctx context.Context, img string) error {
	n.Lock()
	defer n.Unlock()

	log := ctrl.LoggerFrom(ctx, "node", n.name, "action", "remove")

	if err := n.observe(ctx); err != nil {
		return err
	}

	fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
	ref := util.GetImageLabelValue(fqn)

	log.V(4).Info("removing node label", "ref", ref)

	n.removeNodeLabel(coralv1beta1.ImageSyncLabel + "/" + ref)

	return n.update(ctx)
}

func (n *Node) IsReady(ctx context.Context) bool {
	n.Lock()
	defer n.Unlock()

	if err := n.observe(ctx); err != nil {
		return false
	}

	conditionReady := false
	conditionNoDiskPressure := false
	conditionNoPIDPressure := false

	for _, condition := range n.node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			conditionReady = condition.Status == corev1.ConditionTrue
		case corev1.NodeDiskPressure:
			conditionNoDiskPressure = condition.Status == corev1.ConditionFalse
		case corev1.NodePIDPressure:
			conditionNoPIDPressure = condition.Status == corev1.ConditionFalse
		}
	}

	return conditionReady && conditionNoDiskPressure && conditionNoPIDPressure
}

func (n *Node) Matches(ctx context.Context, l []coralv1beta1.NodeSelector) (bool, error) {
	n.Lock()
	defer n.Unlock()

	if err := n.observe(ctx); err != nil {
		return false, err
	}

	ls := labels.NewSelector()
	for _, v := range l {
		req, err := labels.NewRequirement(v.Key, v.Operator, v.Values)
		if err != nil {
			return false, err
		}
		ls = ls.Add(*req)
	}

	return ls.Matches(labels.Set(n.node.GetLabels())), nil
}

func (n *Node) observe(ctx context.Context) error {
	var node corev1.Node
	err := n.client.Get(ctx, client.ObjectKey{Name: n.name}, &node)
	if err != nil {
		return err
	}

	n.node = node.DeepCopy()
	return nil
}

func (n *Node) addNodeLabel(key, value string) {
	l := n.node.GetLabels()
	if l == nil {
		l = make(map[string]string)
	}

	l[key] = value

	n.node.SetLabels(l)
}

func (n *Node) removeNodeLabel(key string) {
	l := n.node.GetLabels()
	if l == nil {
		return
	}

	delete(l, key)
	n.node.SetLabels(l)
}

func (n *Node) update(ctx context.Context) error {
	return n.client.Update(ctx, n.node)
}
