package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gen/pb/gotools/rpc/genrpc/genrpc_proto"

	"github.com/golang/protobuf/proto"
	"github.com/xinlaini/golibs/log"
)

var (
	genBaseDir = filepath.Join(os.Getenv("GOPATH"), "src", "gen", "rpc")
	outDir     = flag.String("out_dir", "", "Output dir relative to $GOPATH/gen/rpc")
)

type data struct {
	ImportProto []string
	SvcDef      *genrpc.Service
}

func main() {
	flag.Parse()
	logger := xlog.NewConsoleLogger()

	const usage = "usage: genrpc [--out_dir=<OUT_DIR>] service_def_file"
	if len(flag.Args()) < 1 {
		logger.Fatal(usage)
	}

	var err error
	text, err := ioutil.ReadFile(flag.Args()[0])
	if err != nil {
		logger.Fatalf("Failed to read %s: ", err)
	}
	dot := &data{SvcDef: &genrpc.Service{}}
	if err = proto.UnmarshalText(string(text), dot.SvcDef); err != nil {
		logger.Fatalf("Failed to unmarshal service definition: %s", err)
	}
	for _, importProto := range dot.SvcDef.ImportProto {
		parts := strings.Split(importProto, ":")
		if len(parts) != 2 {
			logger.Fatalf("Invalid import_proto '%s':", importProto)
		}
		dot.ImportProto = append(dot.ImportProto, filepath.Join("gen", "pb", parts[0], parts[1]))
	}
	dir := filepath.Join(genBaseDir, *outDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Fatalf("Failed to create output dir '%s': %s", dir, err)
	}
	out, err := os.Create(filepath.Join(dir, "rpc_def.go"))
	if err != nil {
		logger.Fatalf("Failed to create file: %s", err)
	}
	defer out.Close()
	t := template.Must(template.New("serviceDef").Parse(tmpl))
	if err = t.Execute(out, dot); err != nil {
		logger.Fatalf("Failed to execute template: %s", err)
	}
	logger.Info("Done")
}
