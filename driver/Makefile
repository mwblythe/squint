mods:
	go get modernc.org/sqlite 
	go get github.com/mattn/go-sqlite3
	go get github.com/go-sql-driver/mysql
	go mod tidy

test: mods
	go test -v -tags "mysql,sqlite" -coverprofile .cover

cover: test
	go tool cover -func .cover

cover_html: test
	go tool cover -html .cover
