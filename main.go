package main

import (
	"ipfs-alpha-entanglement-code/cmd"
	"ipfs-alpha-entanglement-code/util"
)

func main() {
	util.Enable_LogPrint()

	alpha, s, p := 3, 5, 5
	path := "test/data/largeFile.txt"

	err := cmd.Upload(path, alpha, s, p)
	util.CheckError(err, "fail uploading file %s or its entanglement", path)
}
