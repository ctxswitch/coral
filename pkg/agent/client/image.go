package client

import (
	"context"
	"sync"
	"time"

	iutil "ctx.sh/coral/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/cri-client/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)

type Client struct {
	authCache map[string]*crun.AuthConfig
	isc       ImageServiceClient
	rsc       RuntimeServiceClient

	sync.Mutex
}

func New() *Client {
	return &Client{
		authCache: make(map[string]*crun.AuthConfig),
	}
}

func (c *Client) WithImageServiceClient(isc ImageServiceClient) *Client {
	c.isc = isc
	return c
}

func (c *Client) WithRuntimeServiceClient(rsc RuntimeServiceClient) *Client {
	c.rsc = rsc
	return c
}

func (c *Client) Connect(ctx context.Context, addr string) error {
	log := ctrl.LoggerFrom(ctx)

	addr, dialer, err := util.GetAddressAndDialer(addr)
	if err != nil {
		log.Error(err, "get container runtime address failed")
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext( // nolint:staticcheck
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithAuthority("localhost"),
		grpc.WithContextDialer(dialer),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)),
	)
	if err != nil {
		log.Error(err, "connect remote image service failed", "address", addr)
		return err
	}

	c.isc = crun.NewImageServiceClient(conn)
	c.rsc = crun.NewRuntimeServiceClient(conn)

	return nil
}

func (c *Client) Pull(ctx context.Context, fqn string, auth []*crun.AuthConfig) error {
	log := ctrl.LoggerFrom(ctx, "image", fqn)

	if len(auth) == 0 {
		return c.pull(ctx, fqn, nil)
	}

	// TODO: This could cross cache boundaries for the imagesync images causing confusion.  I
	//   was using the imagesync uid, but I don't think the client should be aware of that.
	if auth, ok := c.authCache[fqn]; ok {
		err := c.pull(ctx, fqn, auth)
		if err != nil {
			return err
		}

		log.V(4).Info("failed to pull image with cached credentials")
		delete(c.authCache, fqn)
	}

	for _, a := range auth {
		log.V(4).Info("attempting to pull image with provided credentials")
		err := c.pull(ctx, fqn, a)
		if err == nil {
			log.V(3).Info("successfully pulled image with provided credentials")
			c.authCache[fqn] = a
			return nil
		}
	}

	log.Error(nil, "failed to pull image with provided credentials")
	return ErrUnauthorized
}

func (c *Client) Delete(ctx context.Context, uid, name string) (Info, error) {
	info, err := c.status(ctx, name)
	if err != nil {
		if IsNotFound(err) {
			return Info{}, ErrNotFound
		} else {
			return Info{}, err
		}
	}

	delete(c.authCache, uid)
	return info, nil
}

func (c *Client) Status(ctx context.Context, name string) (Info, error) {
	return c.status(ctx, name)
}

func (c *Client) List(ctx context.Context) ([]string, error) {
	c.Lock()
	defer c.Unlock()

	resp, err := c.isc.ListImages(ctx, &crun.ListImagesRequest{})
	if err != nil {
		return nil, err
	}

	images := make([]string, 0)
	for _, img := range resp.GetImages() {
		images = append(images, img.GetRepoTags()...)
	}

	return images, nil
}

func (c *Client) pull(ctx context.Context, name string, auth *crun.AuthConfig) (err error) {
	c.Lock()
	defer c.Unlock()

	_, err = c.isc.PullImage(ctx, &crun.PullImageRequest{
		Image: &crun.ImageSpec{
			Image: name,
		},
		Auth: auth,
	})

	return err
}

func (c *Client) status(ctx context.Context, name string) (Info, error) {
	c.Lock()
	defer c.Unlock()

	fqn := iutil.GetImageQualifiedName(iutil.DefaultSearchRegistry, name)

	resp, err := c.isc.ImageStatus(ctx, &crun.ImageStatusRequest{
		Image: &crun.ImageSpec{
			Image: fqn,
		},
	})
	if err != nil {
		return Info{}, err
	}

	if resp.GetImage() == nil {
		return Info{}, ErrNotFound
	}

	return Info{
		ID:   resp.GetImage().GetId(),
		Name: fqn,
		Tags: resp.GetImage().GetRepoTags(),
	}, nil
}

var _ ImageClient = &Client{}
