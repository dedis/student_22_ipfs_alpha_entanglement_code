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
	Data         [][]byte
	DataFilter   map[int]struct{}
	Parity       [][][]byte
	ParityFilter []map[int]struct{}
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

	if _, ok := getter.ParityFilter[strand][index-1]; ok {
		err = xerrors.Errorf("no parity exists")
	} else {
		parity = getter.Parity[strand][index-1]
	}

	return
}

// util.Enable_InfoPrint()
// util.Enable_LogPrint()

var alpha, s, p int = 3, 5, 5
var getTest = func(chunkNum int, chunkSize int, missingIndexes map[int]struct{}, missingParities []map[int]struct{}) func(*testing.T) {
	return func(t *testing.T) {
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
		parityChan := make(chan entangler.EntangledBlock, alpha*len(data))
		err := tangler.Entangle(dataChan, parityChan)
		if err != nil {
			t.Fatal("Fail to do entanglement: ", err)
		}
		parities := make([][][]byte, alpha)
		for k := 0; k < alpha; k++ {
			parities[k] = make([][]byte, len(data))
		}
		for parity := range parityChan {
			parities[parity.Strand][parity.LeftBlockIndex-1] = parity.Data
		}

		for len(missingParities) < alpha {
			missingParities = append(missingParities, map[int]struct{}{})
		}

		// create getter
		getter := SimpleGetter{
			Data:         data,
			DataFilter:   missingIndexes,
			Parity:       parities,
			ParityFilter: missingParities}
		util.LogPrint(util.Green("Finish creating getter"))

		lattice := entangler.NewLattice(alpha, s, p, chunkNum, &getter)
		lattice.Init()
		util.LogPrint(util.Green("Finish generating lattice"))

		myData, err := lattice.GetAllData()
		if err != nil {
			t.Fatal("Fail to generate all data: ", err)
		}

		if len(myData) != chunkNum {
			t.Fatalf("Lengths not equal. Got: %d, Expected: %d\n", len(myData), chunkNum)
		}
		for i := 0; i < chunkNum; i++ {
			res := bytes.Compare(myData[i], data[i])
			if res != 0 {
				fmt.Println(string(myData[i]))
				fmt.Println(string(data[i]))
				t.Fatalf("Data not equal. Block Index: %d\n", i+1)
			}
		}
	}
}

func Test_Lattice_No_Recovery(t *testing.T) {
	t.Run("small", getTest(5, 32, map[int]struct{}{}, []map[int]struct{}{}))
	t.Run("medium", getTest(20, 32, map[int]struct{}{}, []map[int]struct{}{}))
}

func Test_Lattice_Single_Data_Lost(t *testing.T) {
	missedFront := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", 0)
		return getTest(chunkNum, chunkSize, map[int]struct{}{0: {}}, []map[int]struct{}{})
	}
	missedEnd := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", chunkNum-1)
		return getTest(chunkNum, chunkSize, map[int]struct{}{chunkNum - 1: {}}, []map[int]struct{}{})
	}
	missedMiddle := func(chunkNum int, chunkSize int) func(*testing.T) {
		missed := 1 + rand.Intn(chunkNum-1)
		util.LogPrint("Missing Position: %d\n", missed)
		return getTest(chunkNum, chunkSize, map[int]struct{}{missed: {}}, []map[int]struct{}{})
	}
	t.Run("middle", missedMiddle(81, 32))
	t.Run("front", missedFront(81, 32))
	t.Run("end", missedEnd(81, 32))
}

func Test_Lattice_Multiple_Data_Lost(t *testing.T) {
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
		return getTest(chunkNum, chunkSize, missedIndexes, []map[int]struct{}{})
	}
	t.Run("2-Miss", missedN(5, 32, 2))
	t.Run("3-Miss", missedN(5, 32, 3))
}

func Test_Lattice_Single_Two_Step_Recovery(t *testing.T) {
	missedFront := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", 0)
		parityMiss := make([]map[int]struct{}, alpha)
		for k := 0; k < alpha; k++ {
			parityMiss[k] = map[int]struct{}{0: {}}
		}
		return getTest(chunkNum, chunkSize, map[int]struct{}{0: {}}, parityMiss)
	}
	missedEnd := func(chunkNum int, chunkSize int) func(*testing.T) {
		util.LogPrint("Missing Position: %d\n", chunkNum-1)
		parityMiss := make([]map[int]struct{}, alpha)
		for k := 0; k < alpha; k++ {
			parityMiss[k] = map[int]struct{}{chunkNum - 1: {}}
		}
		return getTest(chunkNum, chunkSize, map[int]struct{}{chunkNum - 1: {}}, parityMiss)
	}
	missedMiddle := func(chunkNum int, chunkSize int) func(*testing.T) {
		missed := 1 + rand.Intn(chunkNum-1)
		util.LogPrint("Missing Position: %d\n", missed)
		parityMiss := make([]map[int]struct{}, alpha)
		for k := 0; k < alpha; k++ {
			parityMiss[k] = map[int]struct{}{missed: {}}
		}
		return getTest(chunkNum, chunkSize, map[int]struct{}{missed: {}}, parityMiss)
	}
	t.Run("middle", missedMiddle(25, 32))
	t.Run("front", missedFront(25, 32))
	t.Run("end", missedEnd(25, 32))
}

func Test_Lattice_Multiple_Two_Step_Recovery(t *testing.T) {
	missedNM := func(chunkNum int, chunkSize int, missNum int) func(*testing.T) {
		indexes := make([]int, chunkNum)
		for i := 0; i < chunkNum; i++ {
			indexes[i] = i
		}
		missedIndexes := map[int]struct{}{}
		missedParity := map[int]struct{}{}
		for i := 0; i < missNum; i++ {
			r := int(rand.Int63n(int64(len(indexes))))
			missedIndexes[indexes[r]] = struct{}{}
			missedParity[indexes[r]] = struct{}{}
			indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
			indexes = indexes[:len(indexes)-1]
		}
		parityMiss := make([]map[int]struct{}, alpha)
		for k := 0; k < alpha; k++ {
			parityMiss[k] = missedParity
		}
		return getTest(chunkNum, chunkSize, missedIndexes, parityMiss)
	}
	t.Run("2-Miss", missedNM(25, 32, 2))
	t.Run("3-Miss", missedNM(25, 32, 3))
}

func Test_Lattice_Multiple_Random_Lost(t *testing.T) {
	missedN := func(chunkNum int, chunkSize int, missNum int) func(*testing.T) {
		indexes := make([]int, chunkNum*(alpha+1))
		for i := 0; i < len(indexes); i++ {
			indexes[i] = i
		}
		missedIndexes := map[int]struct{}{}
		parityMiss := make([]map[int]struct{}, alpha)
		for k := 0; k < alpha; k++ {
			parityMiss[k] = map[int]struct{}{}
		}

		for i := 0; i < missNum; i++ {
			r := int(rand.Int63n(int64(len(indexes))))

			if r < chunkNum {
				missedIndexes[indexes[r]] = struct{}{}
			} else {
				k := r/chunkNum - 1
				parityMiss[k][r%chunkNum] = struct{}{}
			}

			indexes[r], indexes[len(indexes)-1] = indexes[len(indexes)-1], indexes[r]
			indexes = indexes[:len(indexes)-1]
		}

		return getTest(chunkNum, chunkSize, missedIndexes, parityMiss)
	}
	t.Run("25-Miss", missedN(25, 32, 25))
	t.Run("50-Miss", missedN(25, 32, 50))
}

func Test_Lattice_Whole_Data_Lost(t *testing.T) {
	missedAll := func(chunkNum int, chunkSize int) func(*testing.T) {
		missedIndexes := map[int]struct{}{}
		for i := 1; i < chunkNum; i++ {
			missedIndexes[i] = struct{}{}
		}
		return getTest(chunkNum, chunkSize, missedIndexes, []map[int]struct{}{})
	}
	t.Run("Miss-All", missedAll(10, 32))
}
