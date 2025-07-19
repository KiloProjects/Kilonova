package repository

import "github.com/Masterminds/squirrel"

func init() {
	// Set dollar placeholder format for squirrel
	squirrel.StatementBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
}
