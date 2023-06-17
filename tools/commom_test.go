package tools

import (
	"fmt"
	"testing"
)

func TestGetSessionIdByUserId(t *testing.T) {

	aa := map[int64]bool{}
	for i := 0; i < 5000; i++ {
		id := GetSnowflakeId()
		aa[id] = true
	}
	fmt.Println(len(aa))
}
