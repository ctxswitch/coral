package image

import (
	"context"
	"sync"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	name    string
	node    *Node
	service *Service

	sync.Mutex
}

func New(c client.Client, name string) *Client {
	return &Client{
		name: name,
		node: NewNode(&NodeOptions{
			Client: c,
			Name:   name,
		}),
		service: NewService(),
	}
}

func (c *Client) Connect(ctx context.Context, addr string) error {
	return c.service.Connect(ctx, addr)
}

func (c *Client) Pull(ctx context.Context, id, name, ref string) error {
	c.Lock()
	defer c.Unlock()

	if !c.node.IsReady(ctx) {
		return ErrNodeNotReady
	}

	_, err := c.service.Pull(ctx, id, name)
	if err != nil {
		return err
	}

	if err := c.node.Update(ctx, name, ref); err != nil {
		return err
	}

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
