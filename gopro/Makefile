makepro_proto:
	cd $(PROTOROOT)/makepro && mkdir -p $(GOPATH)/src/gen/pb/makepro/makepro_proto && \
		protoc --proto_path=./ --go_out=$(GOPATH)/src/gen/pb/makepro/makepro_proto *.proto

gopro: makepro_proto
	go build -o ./gopro
