module github.com/kozmod/oniontx/sqlx

go 1.21.0

replace github.com/kozmod/oniontx => ../

require (
	github.com/jmoiron/sqlx v1.3.5
	github.com/kozmod/oniontx v0.0.0
	github.com/lib/pq v1.10.9
)