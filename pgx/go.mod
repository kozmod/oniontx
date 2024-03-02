module github.com/kozmod/oniontx/pgx

go 1.21.0

replace github.com/kozmod/oniontx => ../

require (
	github.com/jackc/pgx/v5 v5.5.3
	github.com/kozmod/oniontx v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
