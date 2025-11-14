package domain

import "errors"

var (
	ErrTeamExists  = errors.New("team already exists")
	ErrPRExists    = errors.New("pull request already exists")
	ErrPRMerged    = errors.New("operation not allowed on merged pull request")
	ErrNotAssigned = errors.New("reviewer is not assigned to this pull request")
	ErrNoCandidate = errors.New("no active replacement candidate available in team")
	ErrNotFound    = errors.New("resource not found")
)
