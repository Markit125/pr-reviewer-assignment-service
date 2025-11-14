package domain

type Team struct {
	TeamName TeamName
	Members  []TeamMember
}

type TeamMember struct {
	UserID   UserID
	Username string
	IsActive bool
}
