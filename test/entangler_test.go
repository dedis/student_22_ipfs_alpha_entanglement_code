package test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
)

func Test_Entanglement(t *testing.T) {
	getTest := func(input string) func(*testing.T) {
		return func(t *testing.T) {
			// util.Enable_LogPrint()

			os.Remove(filepath.Join("data/entangler", input+"_entanglement_0_generated"))
			os.Remove(filepath.Join("data/entangler", input+"_entanglement_1_generated"))
			os.Remove(filepath.Join("data/entangler", input+"_entanglement_2_generated"))

			data, err := os.ReadFile(filepath.Join("data/entangler", input))
			util.CheckError(err, "fail to read input file")

			dataChan := make(chan []byte, len(data)/32+1)

			blockSize := 32
			for i := 0; i < len(data); i += blockSize {
				end := i + blockSize

				if end > len(data) {
					end = len(data)
				}
				dataChan <- data[i:end]
			}
			close(dataChan)

			alpha, s, p := 3, 5, 5
			tangler := entangler.NewEntangler(alpha, s, p)

			outputPaths := make([]string, 3)
			for k := 0; k < alpha; k++ {
				outputPaths[k] = fmt.Sprintf("data/entangler/my_%s_entanglement_%d", input, k)
			}

			parityChan := make(chan entangler.EntangledBlock, alpha*len(data))
			err = tangler.Entangle(dataChan, parityChan)
			if err != nil {
				t.Fail()
				return
			}

			parities := make([][][]byte, alpha)
			for k := 0; k < alpha; k++ {
				parities[k] = make([][]byte, len(data))
			}
			for parity := range parityChan {
				parities[parity.Strand][parity.LeftBlockIndex-1] = parity.Data
			}

			for k := 0; k < alpha; k++ {
				// generate byte array of the current strand
				entangledData := make([]byte, 0)
				for _, parityData := range parities[k] {
					entangledData = append(entangledData, parityData...)
				}

				// write entanglement to file
				err = os.WriteFile(outputPaths[k], entangledData, 0644)
				if err != nil {
					t.Fail()
					return
				}
			}

			for k := 0; k < alpha; k++ {
				expectedResult, err := os.ReadFile(outputPaths[k])
				if err != nil {
					t.Fail()
					return
				}
				myResult, err := os.ReadFile(outputPaths[k])
				if err != nil {
					t.Fail()
					return
				}
				util.CheckError(err, "fail to read horizaontal entanglement file")
				res := bytes.Compare(myResult, expectedResult)
				if res != 0 {
					fmt.Printf(util.Red("Strand %d not equal: %d\n"), k, res)
					t.Fail()
				} else {
					fmt.Printf(util.Green("Strand %d equal\n"), k)
				}
			}
		}
	}

	t.Run("small", getTest("randomSmall"))
	t.Run("median", getTest("randomMedian"))
	t.Run("large", getTest("randomLarge"))
}
