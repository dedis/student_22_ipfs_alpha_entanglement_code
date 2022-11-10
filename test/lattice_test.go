package test

import (
	"bytes"
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	"strings"
	"testing"

	"golang.org/x/xerrors"
)

type SimpleGetter struct {
	entangler.BlockGetter
	Data [][]byte
}

func (getter *SimpleGetter) GetData(index int) (data []byte, err error) {
	if index < 1 || index > len(getter.Data) {
		err = xerrors.Errorf("invalid index")
	} else {
		data = getter.Data[index-1]
	}
	return
}

func Test_Lattice_No_Recovery(t *testing.T) {
	getTest := func(chunkNum int, chunkSize int) func(*testing.T) {
		return func(t *testing.T) {
			data := make([][]byte, 0)
			for i := 0; i < chunkNum; i++ {
				chunk := []byte(strings.Repeat(fmt.Sprintf("%d", i%10), chunkSize))
				data = append(data, chunk)
			}

			getter := SimpleGetter{Data: data}

			alpha, s, p := 3, 5, 5
			lattice := entangler.NewLattice(alpha, s, p, chunkNum, &getter)
			lattice.Init()

			myData, err := lattice.GetAllData()
			if err != nil {
				fmt.Println(err)
				t.Fail()
				return
			}

			if len(myData) != chunkNum {
				fmt.Println(len(myData))
				t.Fail()
				return
			}
			for i := 0; i < chunkNum; i++ {
				res := bytes.Compare(myData[i], data[i])
				if res != 0 {
					t.Fail()
					return
				}
			}
		}
	}
	t.Run("small", getTest(5, 32))
}
