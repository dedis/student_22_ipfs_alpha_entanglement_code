package main

import (
	"ipfs-alpha-entanglement-code/cmd"
	"ipfs-alpha-entanglement-code/util"
)

func main() {
	util.Enable_LogPrint()
	// util.Enable_InfoPrint()

	client, err := cmd.NewClient()
	if err != nil {
		panic(err)
	}

	err = client.Execute()
	if err != nil {
		panic(err)
	}
}
