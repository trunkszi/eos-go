package gosio

import (
	"testing"
	"time"
	"fmt"
)

func TestDeadlineTimer_AsyncWait(t *testing.T) {
	iosv := NewIoContext()
	timer := NewDeadlineTimer(iosv)
	timer.Expires(time.Now().Add(time.Second * 1))

	x := 1
	timer.AsyncWait(func(ec ErrorCode) {
		x = 2
		fmt.Println("hello", time.Now())
		timer.Expires(time.Now().Add(time.Second * 1))
		timer.AsyncWait(func(ec ErrorCode) {
			fmt.Println(x)
			x = 3
			fmt.Println("hi", time.Now())
		})
	})

	iosv.Run()
}