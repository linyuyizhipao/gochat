package tools

import (
	"fmt"
	"testing"
)

func TestGetSessionIdByUserId(t *testing.T) {
	for i := 0; i < 5000; i++ {
		id := GetSnowflakeId()
		fmt.Println(id, "\n ")
	}

}
