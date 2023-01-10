package main

import (
	"fmt"
	"testing"
)

func TestLs(t *testing.T) {
	fmt.Println(GetFileInDir("/home/dong/aaa/webdav"))
}

func TestHumanFileSize(t *testing.T) {
	println(HumanFileSize(7170763))
}
