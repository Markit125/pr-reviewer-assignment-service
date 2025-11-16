package domain

import (
	"time"
)

type PRStatus string

const (
	StatusOpen   PRStatus = "OPEN"
	StatusMerged PRStatus = "MERGED"
)

type PullRequest struct {
	ID                PullRequestID
	Name              string
	AuthorID          UserID
	Status            PRStatus
	AssignedReviewers []UserID
	CreatedAt         time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	ID       PullRequestID
	Name     string
	AuthorID UserID
	Status   PRStatus
}
