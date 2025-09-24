// service/follower_service.go
package service

import (
	"context"
	"database-example/model"
	"database-example/repo"
	"errors"
	"strings"
)

type FollowerService struct {
	FollowerRepo *repo.FollowerRepository
}

var ErrInvalidIDs = errors.New("followerID and followeeID must be non-empty and different")

func (s *FollowerService) Follow(followerID, followeeID string) error {
	// biznis validacija u servis sloju
	followerID = strings.TrimSpace(followerID)
	followeeID = strings.TrimSpace(followeeID)
	if followerID == "" || followeeID == "" || followerID == followeeID {
		return ErrInvalidIDs
	}

	// kreiraj kontekst (isti stil kao u tvom UserService-u)
	ctx := context.Background()
	return s.FollowerRepo.Follow(ctx, followerID, followeeID)
}

func (s *FollowerService) Unfollow(ctx context.Context, followerID, followeeID string) error {
	if followerID == "" || followeeID == "" {
		return errors.New("missing ids")
	}
	if followerID == followeeID {
		return errors.New("cannot unfollow self")
	}
	return s.FollowerRepo.Unfollow(ctx, followerID, followeeID)
}

func (s *FollowerService) GetRecommendations(ctx context.Context, userID string, limit int) ([]model.Recommendation, error) {
	if userID == "" {
		return nil, errors.New("missing user_id")
	}
	if limit <= 0 {
		limit = 10
	}
	return s.FollowerRepo.GetRecommendations(ctx, userID, limit)
}

func (s *FollowerService) GetFollowees(ctx context.Context, userID string, skip, limit int) ([]string, error) {
	if userID == "" {
		return nil, errors.New("missing user_id")
	}
	return s.FollowerRepo.GetFollowees(ctx, userID, skip, limit)
}
