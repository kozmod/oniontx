module github.com/kozmod/oniontx/test

go 1.22

replace (
	github.com/kozmod/oniontx => ../
	github.com/kozmod/oniontx/gorm => ../gorm
	github.com/kozmod/oniontx/pgx => ../pgx
	github.com/kozmod/oniontx/sqlx => ../sqlx
	github.com/kozmod/oniontx/stdlib => ../stdlib
)

require (
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jmoiron/sqlx v1.3.5
	github.com/kozmod/oniontx/gorm v0.3.1
	github.com/kozmod/oniontx/pgx v0.3.1
	github.com/kozmod/oniontx/sqlx v0.3.1
	github.com/kozmod/oniontx/stdlib v0.3.1
	github.com/stretchr/testify v1.9.0
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.7
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kozmod/oniontx v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
