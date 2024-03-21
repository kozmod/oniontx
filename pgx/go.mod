module github.com/kozmod/oniontx/pgx

go 1.20

replace github.com/kozmod/oniontx => ../

require (
	github.com/jackc/pgx/v5 v5.5.5
	github.com/kozmod/oniontx v0.3.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
