package model

import (
	"github.com/google/uuid"
)

type Follow struct {
	ID         uuid.UUID `json:"id"`
	FollowerID string    `json:"followerId"`
	FolloweeID string    `json:"followeeId"`
}

type Recommendation struct {
	UserID string
	Mutual int64
}
