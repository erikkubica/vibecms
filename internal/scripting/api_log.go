package scripting

import (
	"log"

	"github.com/d5/tengo/v2"
)

// logModule returns the cms/log built-in module.
func logModule() map[string]tengo.Object {
	return map[string]tengo.Object{
		"info": &tengo.UserFunction{
			Name: "info",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, nil
				}
				log.Printf("[script] INFO: %s", getString(args[0]))
				return tengo.UndefinedValue, nil
			},
		},
		"warn": &tengo.UserFunction{
			Name: "warn",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, nil
				}
				log.Printf("[script] WARN: %s", getString(args[0]))
				return tengo.UndefinedValue, nil
			},
		},
		"error": &tengo.UserFunction{
			Name: "error",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, nil
				}
				log.Printf("[script] ERROR: %s", getString(args[0]))
				return tengo.UndefinedValue, nil
			},
		},
		"debug": &tengo.UserFunction{
			Name: "debug",
			Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.UndefinedValue, nil
				}
				log.Printf("[script] DEBUG: %s", getString(args[0]))
				return tengo.UndefinedValue, nil
			},
		},
	}
}
