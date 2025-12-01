set CGO_ENABLED=1
set GOOS=linux
set GOARCH=amd64
go build -o build/exam-service ./services/exam-service/cmd/main.go