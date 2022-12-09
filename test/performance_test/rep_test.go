package test

import (
	"ipfs-alpha-entanglement-code/performance"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

var repFactor = 3

func Test_Rep_Only_Data_Loss(t *testing.T) {
	onlyData := func(missNum int, fileinfo performance.FileInfo) func(*testing.T) {
		return func(*testing.T) {
			indexes := make([]int, fileinfo.TotalBlock)
			for i := 0; i < fileinfo.TotalBlock; i++ {
				indexes[i] = i + 1
			}
			missedIndexes := map[int]struct{}{}
			for i := 0; i < missNum; i++ {
				r := int(rand.Int63n(int64(len(indexes))))
				missedIndexes[indexes[r]] = struct{}{}
				indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
				indexes = indexes[:len(indexes)-1]
			}
			result := performance.RepRecover(fileinfo, missedIndexes, nil)
			t.Logf("Data Recovery Rate: %f", result.RecoverRate)
			t.Logf("Successfully Downloaded Block: %d", result.SuccessCnt)
		}
	}

	// missNum: 1
	// Success Data Recovery Rate: 1.000000
	// Success Parity Overhead: 2
	// missNum: 81 (All blocks are missing)
	// Success Data Recovery Rate: 1.000000
	// Success Parity Overhead: 81
	t.Run("test", onlyData(100, performance.InfoMap["25MB"]))
}

func Test_Rep_Only_Parity_Loss(t *testing.T) {
	//var allRates []float32
	//var allOverhead []float32

	onlyParity := func(missNum int, fileinfo performance.FileInfo, iteration int) func(*testing.T) {
		return func(*testing.T) {
			result := performance.RepRecoverWithFilter(fileinfo, missNum, repFactor, iteration)
			require.NoError(t, result.Err)
			t.Logf("Data Recovery Rate: %f", result.RecoverRate)
			t.Logf("Successfully Downloaded Block: %d", result.SuccessCnt)
		}
	}

	//for missingCnt := 0; missingCnt < 312; missingCnt++ {
	//
	//}
	t.Run("test", onlyParity(100, performance.InfoMap["25MB"], 100))
}
