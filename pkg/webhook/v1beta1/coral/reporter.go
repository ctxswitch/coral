package coral

import (
	"context"

	"connectrpc.com/connect"
	coralv1beta1 "ctx.sh/coral/pkg/gen/coral/v1beta1"
	"ctx.sh/coral/pkg/gen/coral/v1beta1/coralv1beta1connect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reporter is the endpoint for nodes to report available images.
func (s *Service) Reporter(ctx context.Context, req *connect.Request[coralv1beta1.ReporterRequest]) (*connect.Response[coralv1beta1.ReporterResponse], error) {
	logger := log.FromContext(ctx,
		"service", coralv1beta1connect.CoralServiceName,
		"path", coralv1beta1connect.CoralServiceReporterProcedure,
	)
	logger.V(6).Info("received request", "request", req)

	s.nodeRef.AddImages(req.Msg.GetNode(), req.Msg.GetImage())

	return connect.NewResponse(&coralv1beta1.ReporterResponse{
		Message: "ok",
		Status:  coralv1beta1.ReporterStatus_OK,
	}), nil
}
