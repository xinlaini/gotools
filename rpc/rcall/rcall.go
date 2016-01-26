package main

import (
	"flag"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
	"github.com/xinlaini/golibs/rpc"
)

func main() {
	flag.Parse()
	logger := xlog.NewPlainLogger()

	const usage = "usage: rcall service::method@addr [request_contents]"

	if len(flag.Args()) != 1 && len(flag.Args()) != 2 {
		logger.Fatal(usage)
	}

	svcAndAddr := strings.Split(flag.Arg(0), "@")
	if len(svcAndAddr) != 2 {
		logger.Fatal(usage)
	}
	svcAndMethod := strings.Split(svcAndAddr[0], "::")
	if len(svcAndMethod) != 2 {
		logger.Fatal(usage)
	}

	ctrl, err := rpc.NewController(rpc.Config{
		Logger: xlog.NewNilLogger(),
	})
	if err != nil {
		logger.Fatalf("Failed to create RPC controller: %s", err)
	}

	client, err := ctrl.NewClient(rpc.ClientOptions{
		ServiceName:  svcAndMethod[0],
		ServiceAddr:  svcAndAddr[1],
		ConnPoolSize: 1,
		Retry:        rpc.DefaultDialRetry,
	})
	if err != nil {
		logger.Fatalf("Failed to create RPC client: %s", err)
	}

	var requestPB *string
	if len(flag.Args()) == 2 {
		s := flag.Arg(1)
		requestPB = &s
	}
	parentCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx := &rpc.ClientContext{Context: parentCtx}
	responsePB, err := client.CallWithTextPB(svcAndMethod[1], ctx, requestPB)
	if err != nil {
		logger.Errorf("Error: %s", err)
	}
	if ctx.Metadata != nil {
		logger.Infof("Response metadata:\n%s", proto.MarshalTextString(ctx.Metadata))
	}
	if responsePB == nil {
		logger.Info("Response: <nil>")
	} else {
		logger.Infof("Response:\n%s", *responsePB)
	}
}
