
fastssz:
	sszgen --path internal/server/structs/structs.go --include ./internal/bls,./internal/bitlist

fix-bls-import:
	# go mod vendor does not import c static files used in herumi
	vend

build-docker:
	docker build -t beacon .

get-spec-tests:
	./scripts/download-spec-tests.sh v1.1.10

protoc:
	protoc --go_out=. --go-grpc_out=. ./internal/server/proto/*.proto

abigen-deposit:
	ethgo abigen --source ./internal/deposit/deposit.abi --package deposit --output ./internal/deposit/
