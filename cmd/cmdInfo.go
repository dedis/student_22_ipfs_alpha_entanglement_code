package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (c *Client) initCmd() {
	c.Command = &cobra.Command{
		Use: "entangler",
	}

	c.AddUploadCmd()
	c.AddDownloadCmd()
}

func (c *Client) AddUploadCmd() {
	var alpha, s, p int
	uploadCmd := &cobra.Command{
		Use:   "upload [path]",
		Short: "Upload a file to IPFS",
		Long:  "Upload a file to IPFS with optional entanglement",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cid, metaCID, err := c.Upload(args[0], alpha, s, p)
			if len(cid) > 0 {
				fmt.Println("Finish adding file to IPFS. File CID: ", cid)
			}
			if len(metaCID) > 0 {
				fmt.Println("Finish adding metaData to IPFS. MetaFile CID: ", metaCID)
			}
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Upload succeeds.")
		},
	}
	uploadCmd.Flags().IntVarP(&alpha, "alpha", "a", 0, "Set entanglement alpha. 0 means no entanglement")
	uploadCmd.Flags().IntVarP(&s, "s", "s", 0, "Set entanglement s")
	uploadCmd.Flags().IntVarP(&p, "p", "p", 0, "Set entanglement p")

	c.AddCommand(uploadCmd)
}

func (c *Client) AddDownloadCmd() {
	var opt DownloadOption
	downloadCmd := &cobra.Command{
		Use:   "download [cid] [path]",
		Short: "Download a file from IPFS",
		Long:  "Download a file from IPFS. Do recovery if data is missing",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			err := c.Download(args[0], args[1], opt)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Download succeeds.")
		},
	}
	downloadCmd.Flags().StringVarP(&opt.MetaCID, "metacid", "m", "", "Provide metafile cid for recovery")
	downloadCmd.Flags().BoolVarP(&opt.UploadRecoverData, "upload-recovery", "u", true, "Allow upload recovered chunk back to IPFS network")
	downloadCmd.Flags().IntSliceVar(&opt.DataFilter, "missing-data", []int{}, "Specify the missing data blocks for testing")

	c.AddCommand(downloadCmd)
}
