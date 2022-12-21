package util

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

func CheckError(err error, msg string, args ...interface{}) {
	if err != nil {
		ThrowError(msg+": %s", args, err)
	}
}

func ThrowError(msg string, args ...interface{}) {
	panic(fmt.Errorf(msg, args...))
}

var GlobalLogPrint = false
var GlobalInfoPrint = false

func EnableLogPrint() {
	GlobalLogPrint = true
}

func DisableLogPrint() {
	GlobalLogPrint = false
}

func EnableInfoPrint() {
	GlobalInfoPrint = true
}

func DisableInfoPrint() {
	GlobalInfoPrint = false
}

func InfoPrintf(format string, a ...interface{}) {
	if !GlobalInfoPrint {
		return
	}
	log.Printf(format, a...)
}

func LogPrintf(format string, a ...interface{}) {
	if !GlobalLogPrint {
		return
	}
	_, file, _, ok := runtime.Caller(1)
	paths := strings.Split(file, "/")
	callerPackage := paths[len(paths)-2]
	if ok {
		format = fmt.Sprintf(White("[%s]: %s\n"), callerPackage, format)
		log.Printf(format, a...)
		return
	}
	log.Printf(format+"\n", a...)
}
