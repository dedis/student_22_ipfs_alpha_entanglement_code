# ipfs-community
Apply alpha entanglement code on IPFS network to improve reliabilities

To uploade files with entanglement (alpha = 3, s = 5, p = 5):
`go run main.go upload <path_to_file> --alpha 3 -s 5 -p 5`

To download files with recovery enable:
`go run main.go download <file_CID> -o <output_path> -m <metadata_CID> -u <enable_missing_block_upload>`

To do performance test:
`go run main.go perf recover -t <test_case> -p <loss_percent_of_parities> -i <iteration>`
`go run main.go perf rep -t <test_case> -p <loss_percent_of_replication> -i <iteration> -r <replication_factor>`
