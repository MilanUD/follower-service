// handlers/follower_handler.go
package handlers

import (
	"context"
	"errors"

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

func (h *FollowerHandler) Unfollow(ctx context.Context, req *followerpb.UnfollowRequest) (*emptypb.Empty, error) {
	if req.GetFollowerId() == "" || req.GetFolloweeId() == "" {
		return nil, status.Error(codes.InvalidArgument, "missing follower_id or followee_id")
	}
	err := h.Svc.Unfollow(ctx, req.GetFollowerId(), req.GetFolloweeId())
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFollowing):
			return nil, status.Error(codes.NotFound, "not following")
		case err.Error() == "cannot unfollow self":
			return nil, status.Error(codes.InvalidArgument, "cannot unfollow self")
		default:
			return nil, status.Errorf(codes.Internal, "unfollow failed: %v", err)
		}
	}
	return &emptypb.Empty{}, nil
}

func (h *FollowerHandler) GetFollowees(ctx context.Context, req *followerpb.GetFolloweesRequest) (*followerpb.GetFolloweesResponse, error) {
	userID := req.GetUserId()
	skip := int(req.GetSkip())
	limit := int(req.GetLimit())

	ids, err := h.Svc.GetFollowees(ctx, userID, skip, limit)
	if err != nil {
		if err.Error() == "missing user_id" {
			return nil, status.Error(codes.InvalidArgument, "missing user_id")
		}
		return nil, status.Errorf(codes.Internal, "get followees failed: %v", err)
	}

	return &followerpb.GetFolloweesResponse{UserIds: ids}, nil
}

func (h *FollowerHandler) GetRecommendations(ctx context.Context, req *followerpb.GetRecommendationsRequest) (*followerpb.GetRecommendationsResponse, error) {
	userID := req.GetUserId()
	limit := int(req.GetLimit())

	recs, err := h.Svc.GetRecommendations(ctx, userID, limit)
	if err != nil {
		if err.Error() == "missing user_id" {
			return nil, status.Error(codes.InvalidArgument, "missing user_id")
		}
		return nil, status.Errorf(codes.Internal, "recommendations failed: %v", err)
	}

	out := &followerpb.GetRecommendationsResponse{
		Items: make([]*followerpb.Recommendation, 0, len(recs)),
	}
	for _, r := range recs {
		out.Items = append(out.Items, &followerpb.Recommendation{
			UserId: r.UserID,
			Mutual: r.Mutual,
		})
	}
	return out, nil
}
