package image

import (
	"context"
	"fmt"
	"time"

	iutil "ctx.sh/coral/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/cri-client/pkg/util"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)

type Service struct {
	collection *Collection
	isc        runtime.ImageServiceClient
	rsc        runtime.RuntimeServiceClient
}

func NewService() *Service {
	return &Service{
		collection: NewCollection(),
	}
}

func (s *Service) Connect(ctx context.Context, addr string) error {
	isc, rsc, err := connectContainerRuntime(ctx, addr)
	if err != nil {
		return err
	}

	s.isc = isc
	s.rsc = rsc
	return nil
}

func (s *Service) Delete(ctx context.Context, id, name string) error {
	log := ctrl.LoggerFrom(ctx)

	info, err := s.status(ctx, name)
	if err != nil && !IsNotFound(err) {
		return fmt.Errorf("failed to get image status: %w", err)
	}

	// TODO: determine whether or not we want to clean up the image.  There's some interesting
	//   issues here with matching the information from the image status to the information provided
	//   by the runtime client.  For now we will just let the kubelet garbage collector handle any
	//   cleanup based on it's GC policy settings.
	s.collection.Remove(id, name, info.ID)

	// TODO: log the references for debugging purposes.
	log.V(3).Info("image", "info", info)

	return nil
}

func (s *Service) Pull(ctx context.Context, id, name string) (Info, error) {
	log := ctrl.LoggerFrom(ctx)

	info, err := s.status(ctx, name)
	if err != nil { // nolint:nestif
		if IsNotFound(err) {
			log.V(3).Info("image not present, pulling", "image", name)
			_, perr := s.isc.PullImage(ctx, &runtime.PullImageRequest{
				Image: &runtime.ImageSpec{
					Image: name,
				},
			})
			if perr != nil {
				return Info{}, perr
			}

			// Retry to get the image status after pulling.  In any case return the error as if it's not
			// found, we'll need to retry the pull.
			info, err = s.status(ctx, name)
			if err != nil {
				log.Error(err, "failed to get image status after pulling")
				return Info{}, err
			}
		} else {
			return Info{}, err
		}
	}

	log.V(3).Info("image present", "image", info.Name)
	s.collection.Add(id, name, info.ID)

	return info, nil
}

func (s *Service) status(ctx context.Context, name string) (Info, error) {
	fqn := iutil.GetImageQualifiedName(iutil.DefaultSearchRegistry, name)

	resp, err := s.isc.ImageStatus(ctx, &runtime.ImageStatusRequest{
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
		ID:         resp.GetImage().GetId(),
		Name:       fqn,
		Tags:       resp.GetImage().GetRepoTags(),
		References: s.collection.References(name, resp.GetImage().GetId()),
	}, nil
}

func connectContainerRuntime(ctx context.Context, addr string) (crun.ImageServiceClient, crun.RuntimeServiceClient, error) {
	addr, dialer, err := util.GetAddressAndDialer(addr)
	if err != nil {
		klog.ErrorS(err, "get container runtime address failed")
		return nil, nil, err
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
		klog.ErrorS(err, "connect remote image service failed", "address", addr)
		return nil, nil, err
	}

	ims := crun.NewImageServiceClient(conn)
	rts := crun.NewRuntimeServiceClient(conn)

	return ims, rts, nil
}
