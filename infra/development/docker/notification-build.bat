set CGO_ENABLED=1
set GOOS=linux
set GOARCH=amd64
go build -o build/notification-service ./services/notification-service/cmd/main.go