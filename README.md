# iptoadder
docker 运行环境试用 编译前执行 go env -w CGO_ENABLED=0

Linux 和 Windows 64位可执行程序
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build main.go


CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build main.go
