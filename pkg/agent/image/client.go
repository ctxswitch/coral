package image

import (
	"context"
	iutil "ctx.sh/coral/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/cri-client/pkg/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sync"
	"time"
)

const (
	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)

type Client struct {
	authCache map[string]*runtime.AuthConfig
	isc       runtime.ImageServiceClient
	rsc       runtime.RuntimeServiceClient

	sync.Mutex
}

func New() *Client {
	return &Client{
		authCache: make(map[string]*runtime.AuthConfig),
	}
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

func (c *Client) Pull(ctx context.Context, uid, name string, auth []*runtime.AuthConfig) (Info, error) {
	log := ctrl.LoggerFrom(ctx, "image", name)

	if len(auth) == 0 {
		err := c.pull(ctx, name, nil)
		if err != nil {
			return Info{}, err
		}
		return c.status(ctx, name)
	}

	if auth, ok := c.authCache[uid]; ok {
		err := c.pull(ctx, name, auth)
		if err == nil {
			return c.status(ctx, name)
		}

		log.V(4).Info("failed to pull image with cached credentials")
		delete(c.authCache, uid)
	}

	for _, a := range auth {
		log.V(4).Info("attempting to pull image with provided credentials")
		err := c.pull(ctx, name, a)
		if err != nil {
			continue
		} else {
			c.authCache[uid] = a
			return c.status(ctx, name)
		}
	}

	log.Error(nil, "failed to pull image with provided credentials", "image", name)
	return Info{}, ErrUnauthorized
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

func (c *Client) pull(ctx context.Context, name string, auth *runtime.AuthConfig) (err error) {
	c.Lock()
	defer c.Unlock()

	_, err = c.isc.PullImage(ctx, &runtime.PullImageRequest{
		Image: &runtime.ImageSpec{
			Image: name,
		},
		Auth: auth,
	})

	return
}

func (c *Client) status(ctx context.Context, name string) (Info, error) {
	c.Lock()
	defer c.Unlock()

	fqn := iutil.GetImageQualifiedName(iutil.DefaultSearchRegistry, name)

	resp, err := c.isc.ImageStatus(ctx, &runtime.ImageStatusRequest{
		Image: &runtime.ImageSpec{
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
