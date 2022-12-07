package test

import (
	"ipfs-alpha-entanglement-code/util"
	"sync"
)

var once sync.Once

func EnableLog(enable bool) {
	once.Do(func() {
		if enable {
			util.Enable_LogPrint()
		}
	})
}
