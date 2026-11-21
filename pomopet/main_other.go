//go:build !windows

package main

import "fmt"

func main() { fmt.Println("PomoPet is a Windows app: build with GOOS=windows (see build.sh).") }
