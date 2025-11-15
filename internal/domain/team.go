package domain

type Team struct {
	Name    TeamName
	Members []TeamMember
}

type TeamMember struct {
	UserID   UserID
	Username string
	IsActive bool
}
