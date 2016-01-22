makepro_proto:
	cd $(PROTOROOT)/makepro && mkdir -p $(GOPATH)/src/gen/pb/makepro && \
		protoc --proto_path=./ --go_out=$(GOPATH)/src/gen/pb/makepro *.proto

gopro: makepro_proto
	go build -o ./gopro
