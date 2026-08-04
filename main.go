package main

import (
	"time"
)

func main() {
	s1 := configNode(":3000", ":8001", []string{})
	s2 := configNode(":4000", ":8002", []string{":3000"})
	s3 := configNode(":7000", ":8003", []string{":3000", ":4000"})

	s1.Start()

	time.Sleep(1 * time.Second)

	go func() {
		s2.Start()
	}()

	time.Sleep(1 * time.Second)

	go func() {
		s3.Start()
	}()

	time.Sleep(1 * time.Second)

	select {}
}
