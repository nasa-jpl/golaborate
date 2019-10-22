.PHONY: superk
.PHONY: envsrv
.PHONY: zygo

superk:
	env GOOS=linux GOARCH=amd64 go build superk/main.go
	scp main misery:~/superk
	rm main

envsrv:
	env GOOS=linux GOARCH=amd64 go build envsrv/main.go
	scp main misery:~/envsrv
	rm main

zygo:
	env GOOS=windows GOARCH=386 go build zygo/main.go
	scp main misery:~/go-zygo
	rm main
