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

			blockSize := 32
			blocks := make([][]byte, 0)
			for i := 0; i < len(data); i += blockSize {
				end := i + blockSize

				if end > len(data) {
					end = len(data)
				}
				blocks = append(blocks, data[i:end])
			}

			tangler := entangler.NewEntangler(3, 5, 5, blockSize, &blocks)
			entanglement := tangler.GetEntanglement()

			for k, parities := range entanglement {
				filename := fmt.Sprintf("%s_entanglement_%d", input, k)
				expectedResult, err := os.ReadFile(filepath.Join("data/entangler", filename))
				util.CheckError(err, "fail to read horizaontal entanglement file")
				res := bytes.Compare(parities, expectedResult)
				if res != 0 {
					fmt.Printf(util.Red("Strand %d not equal: %d\n"), k, res)
					os.WriteFile(filepath.Join("data/entangler", "my_"+filename), parities, 0644)
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
