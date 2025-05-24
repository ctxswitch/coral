package image

import (
	"context"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	name      string
	node      *Node
	service   *Service
	authCache map[string]*runtime.AuthConfig

	sync.Mutex
}

func New(c client.Client, name string) *Client {
	return &Client{
		name: name,
		node: NewNode(&NodeOptions{
			Client: c,
			Name:   name,
		}),
		service:   NewService(),
		authCache: make(map[string]*runtime.AuthConfig),
	}
}

func (c *Client) Connect(ctx context.Context, addr string) error {
	return c.service.Connect(ctx, addr)
}

func (c *Client) Pull(ctx context.Context, id, name, ref string, auth []*runtime.AuthConfig) error {
	c.Lock()
	defer c.Unlock()

	log := ctrl.LoggerFrom(ctx, "image", name, "ref", ref)

	if !c.node.IsReady(ctx) {
		return ErrNodeNotReady
	}

	if len(auth) == 0 {
		return c.pull(ctx, id, name, ref, nil)
	}

	if auth, ok := c.authCache[id]; ok {
		err := c.pull(ctx, id, name, ref, auth)
		if err != nil {
			log.V(4).Info("failed to pull image with cached credentials")
			delete(c.authCache, id)
		}

		return err
	}

	for _, a := range auth {
		log.V(4).Info("attempting to pull image with provided credentials")
		err := c.pull(ctx, id, name, ref, a)
		if err != nil {
			continue
		} else {
			c.authCache[id] = a
			return nil
		}
	}

	log.Error(nil, "failed to pull image with provided credentials", "image", name)
	return nil
}

func (c *Client) Delete(ctx context.Context, id, name, ref string) error {
	c.Lock()
	defer c.Unlock()

	if !c.node.IsReady(ctx) {
		return ErrNodeNotReady
	}

	err := c.node.Remove(ctx, name, ref)
	if err != nil {
		return err
	}

	err = c.service.Delete(ctx, id, name)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) pull(ctx context.Context, id, name, ref string, auth *runtime.AuthConfig) error {
	_, err := c.service.Pull(ctx, id, name, auth)
	if err != nil {
		return err
	}

	if err := c.node.Update(ctx, name, ref); err != nil {
		return err
	}

	return nil
}

func (c *Client) Matches(ctx context.Context, selectors []coralv1beta1.NodeSelector) (bool, error) {
	c.Lock()
	defer c.Unlock()

	return c.node.Matches(ctx, selectors)
}

func (c *Client) Managed(ctx context.Context, id, name string) (bool, error) {
	c.Lock()
	defer c.Unlock()

	return false, nil
}

var _ ImageClient = &Client{}
