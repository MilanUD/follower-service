// handlers/follower_handler.go
package handlers

import (
	"context"

	followerpb "database-example/proto/follower"
	"database-example/repo"
	"database-example/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type FollowerHandler struct {
	followerpb.UnimplementedFollowerServiceServer
	Svc *service.FollowerService
}

func NewFollowerHandler(svc *service.FollowerService) *FollowerHandler {
	return &FollowerHandler{Svc: svc}
}

// Opcioni health check da probaš konekciju (ako imaš service.Ping, pozovi njega)
func (h *FollowerHandler) Ping(ctx context.Context, _ *followerpb.PingRequest) (*followerpb.PingResponse, error) {
	return &followerpb.PingResponse{Message: "pong"}, nil
}

func (h *FollowerHandler) Follow(ctx context.Context, req *followerpb.FollowRequest) (*emptypb.Empty, error) {
	if err := h.Svc.Follow(req.FollowerId, req.FolloweeId); err != nil {
		switch err {
		case service.ErrInvalidIDs:
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case repo.ErrUserNotFound:
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, "db error")
		}
	}
	return &emptypb.Empty{}, nil
}
