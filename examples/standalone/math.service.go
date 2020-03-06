package main

import (
	"fmt"

	"github.com/nnqq/moleculer"
	"github.com/nnqq/moleculer/broker"
	"github.com/nnqq/moleculer/payload"
)

var mathService = moleculer.ServiceSchema{
	Name: "math",
	Actions: []moleculer.Action{
		{
			Name: "add",
			Handler: func(ctx moleculer.Context, params moleculer.Payload) interface{} {
				return params.Get("a").Int() + params.Get("b").Int()
			},
		},
	},
}

func main() {
	var bkr = broker.New(&moleculer.Config{LogLevel: "error"})
	bkr.Publish(mathService)
	bkr.Start()
	result := <-bkr.Call("math.add", payload.New(map[string]int{
		"a": 10,
		"b": 130,
	}))
	fmt.Println("result: ", result.Int()) //$ result: 140
	bkr.Stop()
}
