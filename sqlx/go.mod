module github.com/kozmod/oniontx/sqlx

go 1.20

replace github.com/kozmod/oniontx => ../

require (
	github.com/jmoiron/sqlx v1.3.5
	github.com/kozmod/oniontx v0.3.10
	github.com/lib/pq v1.10.9
)

require github.com/go-sql-driver/mysql v1.8.0 // indirect
