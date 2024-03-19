package jq

import (
	"context"
	"fmt"
	"log"

	"github.com/itchyny/gojq"
	"mvdan.cc/sh/v3/interp"
)

type Builtin struct{}

func (c *Builtin) ProvideExecHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if args[0] == "jq" {
			// fmt.Println(args[1])
			query, err := gojq.Parse(args[1])
			if err != nil {
				log.Fatalln(err)
			}
			input := map[string]any{"foo": []any{1, 2, 3}}
			iter := query.Run(input) // or query.RunWithContext
			for {
				v, ok := iter.Next()
				if !ok {
					break
				}
				if err, ok := v.(error); ok {
					log.Fatalln(err)
				}
				fmt.Printf("%#v\n", v)
			}
			return nil
		}
		return next(ctx, args)
	}
}
