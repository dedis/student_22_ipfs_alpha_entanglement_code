package test

import (
	"ipfs-alpha-entanglement-code/performance"
	"math/rand"
	"testing"
)

func Test_Only_Data_Loss(t *testing.T) {
	// util.Enable_LogPrint()
	onlyData := func(missNum int, fileinfo performance.FileInfo) func(*testing.T) {
		return func(*testing.T) {
			indexes := make([]int, fileinfo.TotalBlock)
			for i := 0; i < fileinfo.TotalBlock; i++ {
				indexes[i] = i
			}
			missedIndexes := map[int]struct{}{}
			for i := 0; i < missNum; i++ {
				r := int(rand.Int63n(int64(len(indexes))))
				missedIndexes[indexes[r]] = struct{}{}
				indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
				indexes = indexes[:len(indexes)-1]
			}
			result := performance.Recovery(fileinfo, missedIndexes, nil)
			t.Logf("Data Recovery Rate: %f", result.RecoverRate)
			t.Logf("Parity Overhead: %d", result.DownloadParity)
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

func Test_Only_Parity_Loss(t *testing.T) {
	//var allRates []float32
	//var allOverhead []float32
	var accuRate float32
	var accuOverhead uint
	var accuSuccessCnt int

	onlyParity := func(missNum int, fileinfo performance.FileInfo) func(*testing.T) {
		return func(*testing.T) {
			indexes := make([][]int, 3)
			for i := range indexes {
				indexes[i] = make([]int, fileinfo.TotalBlock)
			}
			for i := range indexes {
				for j := 0; j < fileinfo.TotalBlock; j++ {
					indexes[i][j] = j + 1
				}
			}

			/* All data block is missing */
			missedDataIndexes := map[int]struct{}{}
			for i := 0; i < fileinfo.TotalBlock; i++ {
				missedDataIndexes[i] = struct{}{}
			}

			/* Some parity block is missing */
			missedParityIndexes := []map[int]struct{}{{}, {}, {}}
			for i := 0; i < missNum; i++ {
				rOuter := int(rand.Int63n(int64(3)))
				for len(indexes[rOuter]) == 0 {
					rOuter = int(rand.Int63n(int64(3)))
				}
				rInner := int(rand.Int63n(int64(len(indexes[rOuter]))))
				missedParityIndexes[rOuter][indexes[rOuter][rInner]] = struct{}{}
				indexes[rOuter][rInner], indexes[rOuter][len(indexes[rOuter])-1] =
					indexes[rOuter][len(indexes[rOuter])-1], indexes[rOuter][rInner]
				indexes[rOuter] = indexes[rOuter][:len(indexes[rOuter])-1]
			}
			result := performance.Recovery(fileinfo, missedDataIndexes, missedParityIndexes)
			t.Logf("Data Recovery Rate: %f", result.RecoverRate)
			t.Logf("Parity Overhead: %d", result.DownloadParity)
			t.Logf("Successfully Downloaded Block: %d", result.SuccessCnt)
			accuRate += result.RecoverRate
			accuOverhead += result.DownloadParity
			accuSuccessCnt += result.SuccessCnt
		}
	}

	//for missingCnt := 0; missingCnt < 312; missingCnt++ {
	//
	//}
	t.Run("test", onlyParity(100, performance.InfoMap["25MB"]))
}
