package test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ipfs-alpha-entanglement-code/entangler"

	"github.com/stretchr/testify/require"
)

func Test_Entanglement(t *testing.T) {
	EnableLog(true)
	getTest := func(input string) func(*testing.T) {
		return func(t *testing.T) {
			os.Remove(filepath.Join("../data/entangler", input+"_entanglement_0_generated"))
			os.Remove(filepath.Join("../data/entangler", input+"_entanglement_1_generated"))
			os.Remove(filepath.Join("../data/entangler", input+"_entanglement_2_generated"))

			data, err := os.ReadFile(filepath.Join("../data/entangler", input))
			require.NoError(t, err)

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
				outputPaths[k] = fmt.Sprintf("../data/entangler/my_%s_entanglement_%d", input, k)
				defer os.Remove(outputPaths[k])
			}

			parityChan := make(chan entangler.EntangledBlock, alpha*len(data))
			err = tangler.Entangle(dataChan, parityChan)
			require.NoError(t, err)

			err = tangler.WriteEntanglementToFile(0, outputPaths, parityChan)
			require.NoError(t, err)

			for k := 0; k < alpha; k++ {
				expectedResult, err := os.ReadFile(outputPaths[k])
				require.NoError(t, err)

				myResult, err := os.ReadFile(outputPaths[k])
				require.NoError(t, err)

				res := bytes.Compare(myResult, expectedResult)
				require.Equal(t, res, 0)
			}
		}
	}

	t.Run("small", getTest("randomSmall"))
	t.Run("median", getTest("randomMedian"))
	t.Run("large", getTest("randomLarge"))
}
