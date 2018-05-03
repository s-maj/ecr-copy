# ECR copy

This tool allows to copy AWS ECR repositories and their content between AWS regions.

## Getting Started

### Prerequisites
* Go 1.10 or later
* Dep 0.4.1 or later

### Installing
* `dep ensure` 
* `go build main.go` or for cross compile `GOOS=linux GOARCH=amd64 go build main.go`

### Usage
main copy -s source -d destination (`./main copy -s us-west-2 -d eu-west-2`)