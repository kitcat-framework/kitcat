module github.com/expectedsh/kitcat/pkg/kitpg

go 1.21.2

require (
	github.com/expectedsh/kitcat v0.0.0
	github.com/jackc/pgx/v5 v5.4.3
	github.com/samber/lo v1.38.1
	gorm.io/datatypes v1.2.0
	gorm.io/driver/postgres v1.5.3
	gorm.io/gorm v1.25.4
)

require (
	github.com/expectedsh/dig v0.0.1-expected // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/text v0.13.0 // indirect
	gorm.io/driver/mysql v1.4.7 // indirect
)

replace github.com/expectedsh/kitcat => ../../
