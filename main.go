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

	client.Execute()

	// defer func() { //catch or finally
	// 	if err := recover(); err != nil { //catch
	// 		fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
	// 		// os.Exit(1)
	// 	}
	// }()

	// alpha, s, p := 3, 5, 5
	// path := "test/data/largeFile.txt"
	// outpath := "test/data/downloaded_largeFile.txt"

	// client, err := cmd.NewClient()
	// util.CheckError(err, "fail to create client: ", path)
	// fileCID, metaCID, err := client.Upload(path, alpha, s, p)
	// util.CheckError(err, "fail uploading file %s or its entanglement", path)

	// dataFilter := []int{2, 5}
	// fmt.Println("Pretend Block loss: DataBlock(Index=1) and DataBlock(Index=3)")

	// option := cmd.DownloadOption{
	// 	MetaCID:           metaCID,
	// 	UploadRecoverData: true,
	// 	DataFilter:        dataFilter,
	// }
	// err = client.Download(fileCID, outpath, option)
	// util.CheckError(err, "fail downloading file %s", path)

	// cmdExecutor := exec.Command("diff", path, outpath)
	// stdout, err := cmdExecutor.Output()
	// util.CheckError(err, "fail to check differences between original and recovered files")
	// if len(stdout) > 0 {
	// 	fmt.Printf("Verifier: %s and %s differ\n", path, outpath)
	// } else {
	// 	fmt.Printf("Verifier: %s and %s are the same\n", path, outpath)
	// }
}
