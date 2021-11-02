test:
	go test -v -tags "mysql,sqlite" -coverprofile .cover

cover: test
	go tool cover -func .cover

cover_html: test
	go tool cover -html .cover
