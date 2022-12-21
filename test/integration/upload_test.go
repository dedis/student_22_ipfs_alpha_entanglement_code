package integration

import (
	"fmt"
	"ipfs-alpha-entanglement-code/cmd"
	"ipfs-alpha-entanglement-code/performance"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Upload(t *testing.T) {
	// util.EnableLogPrint()
	alpha, s, p := 3, 5, 5
	upload := func(filepath string, expectedCID string, expectedMetaCID string) func(*testing.T) {
		return func(t *testing.T) {
			client, err := cmd.NewClient()
			require.NoError(t, err)

			rootCID, metaCID, pinResult, err := client.Upload(filepath, alpha, s, p)
			require.NoError(t, err)

			require.Equal(t, expectedCID, rootCID)
			require.Equal(t, expectedMetaCID, metaCID)

			err = pinResult()
			require.NoError(t, err)
		}
	}

	for _, testcase := range []string{"5MB", "10MB", "20MB", "25MB"} {
		filepath := fmt.Sprintf("../data/largefile_%s.txt", testcase)
		info := performance.InfoMap[testcase]
		t.Run(testcase, upload(filepath, info.FileCID, info.MetaCID))
	}
}
