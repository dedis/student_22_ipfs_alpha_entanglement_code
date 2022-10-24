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
			h, r, l := tangler.GetEntanglement()

			myResultH := make([]byte, 0)
			// fmt.Print(util.Yellow("H: "))
			for _, block := range h {
				// fmt.Printf(util.Yellow("(%d, %d) "), block.LeftBlockIndex, block.RightBlockIndex)
				myResultH = append(myResultH, block.Data...)
			}
			expectedResultH, err := os.ReadFile(filepath.Join("data/entangler", input+"_entanglement_0"))
			util.CheckError(err, "fail to read horizaontal entanglement file")
			res := bytes.Compare(myResultH, expectedResultH)
			if res != 0 {
				fmt.Printf(util.Red("Horizontal Strand not equal: %d\n"), res)
				os.WriteFile(filepath.Join("data/entangler", input+"_entanglement_0_generated"), myResultH, 0644)
				t.Fail()
			} else {
				fmt.Println(util.Green("Horizontal Strand equal"))
			}

			myResultRH := make([]byte, 0)
			// fmt.Print(util.Yellow("RH: "))
			for _, block := range r {
				// fmt.Printf(util.Yellow("(%d, %d) "), block.LeftBlockIndex, block.RightBlockIndex)
				myResultRH = append(myResultRH, block.Data...)
			}
			expectedResultRH, err := os.ReadFile(filepath.Join("data/entangler", input+"_entanglement_1"))
			util.CheckError(err, "fail to read right-hand entanglement file")
			res = bytes.Compare(myResultRH, expectedResultRH)
			if res != 0 {
				fmt.Printf(util.Red("Right-Hand Strand not equal: %d\n"), res)
				os.WriteFile(filepath.Join("data/entangler", input+"_entanglement_1_generated"), myResultH, 0644)
				t.Fail()
			} else {
				fmt.Println(util.Green("Right-Hand Strand equal"))
			}

			myResultLH := make([]byte, 0)
			// fmt.Print(util.Yellow("LH: "))
			for _, block := range l {
				// fmt.Printf(util.Yellow("(%d, %d) "), block.LeftBlockIndex, block.RightBlockIndex)
				myResultLH = append(myResultLH, block.Data...)
			}
			expectedResultLH, err := os.ReadFile(filepath.Join("data/entangler", input+"_entanglement_2"))
			util.CheckError(err, "fail to read left-hand entanglement file")
			res = bytes.Compare(myResultLH, expectedResultLH)
			if res != 0 {
				fmt.Printf(util.Red("Left-Hand Strand not equal: %d\n"), res)
				os.WriteFile(filepath.Join("data/entangler", input+"_entanglement_2_generated"), myResultH, 0644)
				t.Fail()
			} else {
				fmt.Println(util.Green("Left-Hand Strand equal\n"))
			}
		}
	}

	t.Run("small", getTest("randomSmall"))
	t.Run("median", getTest("randomMedian"))
	t.Run("large", getTest("randomLarge"))
}
