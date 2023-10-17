module github.com/expectedsh/kitcat/pkg/kitsqlite

go 1.21.2

require (
	github.com/expectedsh/kitcat v0.0.0
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.4
)

require (
	github.com/expectedsh/dig v0.0.1-expected // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.17 // indirect
)

replace github.com/expectedsh/kitcat => ../../
