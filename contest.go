package kilonova

import "time"

type Contest struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`

	Name        string `json:"name"`
	Description string `json:"description"`

	Published bool      `json:"published"`
	Endless   bool      `json:"endless"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`

	Problems []*Problem `json:"problems"`
	Authors  []*User    `json:"authors"`
}
