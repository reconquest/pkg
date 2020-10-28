package assert

import (
	"fmt"
)

func NotZero(i int, format string, values ...interface{}) {
	if i == 0 {
		panic(fmt.Errorf(format, values...))
	}
}

func NotZero64(i int64, format string, values ...interface{}) {
	if i == 0 {
		panic(fmt.Errorf(format, values...))
	}
}

func NotNil(obj interface{}, format string, values ...interface{}) {
	if obj == nil {
		panic(fmt.Errorf(format, values...))
	}
}
