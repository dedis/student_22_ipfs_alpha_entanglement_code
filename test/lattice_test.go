package test

import (
	"bytes"
	"fmt"
	"ipfs-alpha-entanglement-code/entangler"
	"ipfs-alpha-entanglement-code/util"
	"math/rand"
	"strings"
	"testing"

	"golang.org/x/xerrors"
)

type SimpleGetter struct {
	entangler.BlockGetter
	Data       [][]byte
	DataFilter map[int]struct{}
	Parity     [][][]byte
}

func (getter *SimpleGetter) GetData(index int) (data []byte, err error) {
	if index < 1 || index > len(getter.Data) {
		err = xerrors.Errorf("invalid index")
	} else {
		if _, ok := getter.DataFilter[index-1]; ok {
			err = xerrors.Errorf("no data exists")
		} else {
			data = getter.Data[index-1]
		}
	}
	return
}

func (getter *SimpleGetter) GetParity(index int, strand int) (parity []byte, err error) {
	if index < 1 || index > len(getter.Data) {
		err = xerrors.Errorf("invalid index")
		return
	}
	if strand < 0 || strand > len(getter.Parity) {
		err = xerrors.Errorf("invalid strand")
		return
	}
	parity = getter.Parity[strand][index-1]
	if len(parity) == 0 {
		err = xerrors.Errorf("no parity exists")
	}

	return
}

var getTest = func(chunkNum int, chunkSize int, missingIndexes map[int]struct{}) func(*testing.T) {
	return func(t *testing.T) {
		alpha, s, p := 3, 5, 5

		// generate data
		data := make([][]byte, 0)
		for i := 0; i < chunkNum; i++ {
			chunk := []byte(strings.Repeat(fmt.Sprintf("%d", i%10), chunkSize))
			data = append(data, chunk)
		}

		// generate parity
		tangler := entangler.NewEntangler(alpha, s, p)
		dataChan := make(chan []byte, len(data))
		for _, chunk := range data {
			dataChan <- chunk
		}
		close(dataChan)
		err := tangler.Entangle(dataChan)
		if err != nil {
			t.Fail()
			return
		}
		parity := make([][][]byte, alpha)
		for k, strandBlocks := range tangler.ParityBlocks {
			parity[k] = make([][]byte, len(strandBlocks))
			for i, block := range strandBlocks {
				parity[k][i] = append(parity[k][i], block.Data...)
			}
		}

		// create getter
		getter := SimpleGetter{
			Data:       data,
			DataFilter: missingIndexes,
			Parity:     parity}
		util.LogPrint(util.Green("Finish creating getter"))

		lattice := entangler.NewLattice(alpha, s, p, chunkNum, &getter)
		lattice.Init()

		util.LogPrint(util.Green("Finish generating lattice"))
		myData, err := lattice.GetAllData()
		if err != nil {
			fmt.Println(err)
			t.Fail()
			return
		}

		if len(myData) != chunkNum {
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

func Test_Lattice_No_Recovery(t *testing.T) {
	t.Run("small", getTest(5, 32, map[int]struct{}{}))
	t.Run("medium", getTest(20, 32, map[int]struct{}{}))
}

func Test_Lattice_Single_Data_Lost(t *testing.T) {
	util.Enable_LogPrint()
	// util.Enable_InfoPrint()
	missedFront := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", 0)
		return getTest(chunkNum, chunkSize, map[int]struct{}{0: {}})
	}
	missedEnd := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", chunkNum-1)
		return getTest(chunkNum, chunkSize, map[int]struct{}{chunkNum - 1: {}})
	}
	missedMiddle := func(chunkNum int, chunkSize int) func(*testing.T) {
		missed := 1 + rand.Intn(chunkNum-1)
		util.LogPrint("Missing Position: %d\n", missed)
		return getTest(chunkNum, chunkSize, map[int]struct{}{missed: {}})
	}
	t.Run("middle", missedMiddle(5, 32))
	t.Run("front", missedFront(5, 32))
	t.Run("end", missedEnd(5, 32))
}

func Test_Lattice_Multiple_Data_Lost(t *testing.T) {
	util.Enable_LogPrint()
	// util.Enable_InfoPrint()
	missedN := func(chunkNum int, chunkSize int, missNum int) func(*testing.T) {
		indexes := make([]int, chunkNum)
		for i := 0; i < chunkNum; i++ {
			indexes[i] = i
		}
		missedIndexes := map[int]struct{}{}
		for i := 0; i < missNum; i++ {
			r := int(rand.Int63n(int64(len(indexes))))
			missedIndexes[indexes[r]] = struct{}{}
			indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
			indexes = indexes[:len(indexes)-1]
		}
		return getTest(chunkNum, chunkSize, missedIndexes)
	}
	t.Run("2-Miss", missedN(5, 32, 2))
	t.Run("3-Miss", missedN(5, 32, 3))
}
