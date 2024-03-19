package golang

import (
	"context"
	"fmt"
	"log"
	"path"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/stdlib"
	"mvdan.cc/sh/v3/interp"
)

type Builtin struct {
	imports map[string]map[string]reflect.Value
}

func (c *Builtin) Initialize() {
	c.imports = make(map[string]map[string]reflect.Value)
}

func (c *Builtin) ProvideExecHandler(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		if args[0] == "imports" {
			for k, _ := range c.imports {
				fmt.Println(k)
			}
			return nil
		}
		if args[0] == "import" {
			for k, v := range stdlib.Symbols {
				if args[1] == path.Dir(k) {
					c.imports[path.Base(k)] = v
					break
				}
			}
			return nil
		}
		if strings.Contains(args[0], ".") {
			parts := strings.Split(args[0], ".")
			if i, ok := c.imports[parts[0]]; ok {
				if f, ok := i[parts[1]]; ok {
					var vargs []any
					for _, a := range args {
						vargs = append(vargs, a)
					}
					ret, err := Call(f, vargs[1:])
					if err != nil {
						log.Fatal(err)
					}
					for _, v := range ret {
						fmt.Println(v)
					}
					return nil
				}
			}
		}
		return next(ctx, args)
	}
}
