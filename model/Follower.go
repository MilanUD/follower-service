package model

import (
	"github.com/google/uuid"
)

type Follow struct {
	ID         uuid.UUID `json:"id"`
	FollowerID string    `json:"followerId"`
	FolloweeID string    `json:"followeeId"`
}
