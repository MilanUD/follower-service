// service/follower_service.go
package service

import (
	"context"
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
