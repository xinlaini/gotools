package main

const tmpl = `package {{.SvcDef.Package}}

import "reflect"
{{range .ImportProto}}import "{{.}}"
{{end}}import "github.com/xinlaini/golibs/rpc"
{{with .SvcDef}}
type {{.ServiceName}}Service interface {
{{range .Method}}	{{.Name}}(ctx *rpc.ServerContext, req *{{.RequestProto}}) (*{{.ResponseProto}}, error)
{{end}}}

type {{.ServiceName}}Client struct {
	rpcClient *rpc.Client
}

func New{{.ServiceName}}Client(ctrl *rpc.Controller, opts rpc.ClientOptions) (*{{.ServiceName}}Client, error) {
	rpcClient, err := ctrl.NewClient(opts)
	if err != nil {
		return nil, err
	}
	return &{{.ServiceName}}Client{rpcClient: rpcClient}, nil
}
{{with $root := .}}{{range .Method}}
func (c *{{$root.ServiceName}}Client) {{.Name}}(ctx *rpc.ClientContext, req *{{.RequestProto}}) (*{{.ResponseProto}}, error) {
	pbResp, err := c.rpcClient.Call("{{.Name}}", ctx, req, reflect.TypeOf((*{{.ResponseProto}})(nil)).Elem())
	if err != nil {
		return nil, err
	}
	if pbResp == nil {
		return nil, nil
	}
	return pbResp.(*{{.ResponseProto}}), nil
}
{{end}}
func (c* {{$root.ServiceName}}Client) Close() {
	c.rpcClient.Close()
}{{end}}{{end}}
`
