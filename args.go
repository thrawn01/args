package args

import (
	"fmt"
)

type Parse struct {
	thing string
}

type OptRule func(*Parse)

func Parser() Parse {
	return Parse{}
}

func (self *Parse) Opt(optName string, rules ...OptRule) {
	for _, rule := range rules {
		rule(self)
	}
}

func Alias(optName string) OptRule {
	return func(parse *Parse) {
		fmt.Printf("Alias(%s)", optName)
	}
}
