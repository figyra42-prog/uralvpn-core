// Command uralvpn_merge merges a sing-box profile and an Xray subscription (UralVPN CI tests).
package main

import (
	"fmt"
	"os"

	"github.com/sagernet/sing-box/common/uralvpn"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: uralvpn_merge <sing-box.json> <xray.json>")
		os.Exit(2)
	}
	singbox, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	xray, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	merged, err := uralvpn.Merge(string(singbox), string(xray))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.WriteString(merged)
}
