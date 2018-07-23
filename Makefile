all:
	GOOS=darwin go build -o goku-darwin-amd64
	GOOS=windows go build -o goku-windows-amd64.exe
	GOOS=linux go build -o goku-linux-amd64

clean:
	rm goku-*

