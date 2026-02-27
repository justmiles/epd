build:
	go build -o epd .

proto:
	protoc --go_out=proto/epdpb --go_opt=paths=source_relative \
		--go-grpc_out=proto/epdpb --go-grpc_opt=paths=source_relative \
		-I proto proto/epd.proto
