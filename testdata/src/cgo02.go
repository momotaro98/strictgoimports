package src

import (
	_ "github.com/golang/mock/gomock"
	_ "github.com/momotaro98/strictgoimports"

	_ "fmt"
)

/*
#include <math.h>
#cgo LDFLAGS: -lm
*/
import "C"
