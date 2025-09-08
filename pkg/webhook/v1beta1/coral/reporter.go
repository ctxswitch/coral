// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
