package cmd

import (
	"fmt"
	"ipfs-alpha-entanglement-code/performance"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// initCmd inits cmd for user interaction
func (c *Client) initCmd() {
	c.Command = &cobra.Command{
		Use: "entangler",
	}

	c.AddUploadCmd()
	c.AddDownloadCmd()
	c.AddPerformanceCmd()
}

// AddUploadCmd enables upload functionality
func (c *Client) AddUploadCmd() {
	var alpha, s, p int
	uploadCmd := &cobra.Command{
		Use:   "upload [path]",
		Short: "Upload a file to IPFS",
		Long:  "Upload a file to IPFS with optional entanglement",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cid, metaCID, pinResult, err := c.Upload(args[0], alpha, s, p)
			if len(cid) > 0 {
				fmt.Println("Finish adding file to IPFS. File CID: ", cid)
			}
			if len(metaCID) > 0 {
				fmt.Println("Finish adding metaData to IPFS. MetaFile CID: ", metaCID)
			}
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			if pinResult != nil {
				err = pinResult()
				if err != nil {
					fmt.Println("Error:", err)
					os.Exit(1)
				}
			}
			fmt.Println("Upload succeeds.")
		},
	}
	uploadCmd.Flags().IntVarP(&alpha, "alpha", "a", 0, "Set entanglement alpha. 0 means no entanglement")
	uploadCmd.Flags().IntVarP(&s, "s", "s", 0, "Set entanglement s")
	uploadCmd.Flags().IntVarP(&p, "p", "p", 0, "Set entanglement p")

	c.AddCommand(uploadCmd)
}

// AddDownloadCmd enables download functionality
func (c *Client) AddDownloadCmd() {
	var opt DownloadOption
	var path string
	downloadCmd := &cobra.Command{
		Use:   "download [cid] [path]",
		Short: "Download a file from IPFS",
		Long:  "Download a file from IPFS. Do recovery if data is missing",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			out, err := c.Download(args[0], path, opt)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}
			fmt.Printf("Download succeeds to '%s'.\n", out)
		},
	}
	downloadCmd.Flags().StringVarP(&path, "output", "o", "", "Provide output path to store the downloaded stuff")
	downloadCmd.Flags().StringVarP(&opt.MetaCID, "metacid", "m", "", "Provide metafile cid for recovery")
	downloadCmd.Flags().BoolVarP(&opt.UploadRecoverData, "upload-recovery", "u", true, "Allow upload recovered chunk back to IPFS network")
	downloadCmd.Flags().IntSliceVar(&opt.DataFilter, "missing-data", []int{}, "Specify the missing data blocks for testing")

	c.AddCommand(downloadCmd)
}

func (c *Client) AddPerformanceCmd() {
	var rootCmd = &cobra.Command{Use: "perf"}

	var fileCase string
	var lossPercent float32
	var iteration int
	recoverCmd := &cobra.Command{
		Use:   "recover [testcase] [loss-percentage]",
		Short: "Performance test for block recovery",
		Long:  "Performance test for block recovery during download from IPFS",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			rand.Seed(time.Now().UnixNano())
			result := performance.Perf_Recovery(fileCase, lossPercent, iteration)
			if result.Err != nil {
				fmt.Println("Error:", result.Err)
				return
			}
			fmt.Printf("Data Recovery Rate: %f\n", result.RecoverRate)
			fmt.Printf("Parity Overhead: %f\n", result.DownloadParity)
			fmt.Printf("Successfully Downloaded Block: %d\n", result.PartialSuccessCnt)
		},
	}
	recoverCmd.Flags().StringVarP(&fileCase, "testcase", "t", "25MB", "Test cases of different file sizes")
	recoverCmd.Flags().Float32VarP(&lossPercent, "loss-percent", "p", 0.5, "Loss percentage of the parities")
	recoverCmd.Flags().IntVarP(&iteration, "iteration", "i", 5, "Repeat the performance test for several times")
	rootCmd.AddCommand(recoverCmd)

	var repFactor int
	repCmd := &cobra.Command{
		Use:   "rep [testcase] [loss-percentage]",
		Short: "Performance test for blocks replication",
		Long:  "Performance test for blocks that are replicated in the IPFS",
		Args:  cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			rand.Seed(time.Now().UnixNano())
			result := performance.Perf_Replication(fileCase, lossPercent, repFactor, iteration)
			if result.Err != nil {
				fmt.Println("Error:", result.Err)
				return
			}
			fmt.Printf("Data Recovery Rate: %f\n", result.RecoverRate)
			// fmt.Printf("Parity Overhead: %d\n", result.DownloadParity)
			fmt.Printf("Successfully Downloaded Block: %d\n", result.PartialSuccessCnt)
		},
	}
	repCmd.Flags().StringVarP(&fileCase, "testcase", "t", "25MB", "Test cases of different file sizes")
	repCmd.Flags().Float32VarP(&lossPercent, "loss-percent", "p", 0.5, "Loss percentage of the replication")
	repCmd.Flags().IntVarP(&iteration, "iteration", "i", 5, "Repeat the performance test for several times")
	repCmd.Flags().IntVarP(&repFactor, "rep-factor", "r", 3, "Set the replication factor of the data")
	rootCmd.AddCommand(repCmd)

	c.AddCommand(rootCmd)
}
