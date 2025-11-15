package domain

type User struct {
	ID       UserID
	Username string
	TeamName TeamName
	IsActive bool
}
