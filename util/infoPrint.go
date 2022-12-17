package util

import (
	"fmt"
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

var Global_LogPrint bool = false
var Global_InfoPrint bool = false

func Enable_LogPrint() {
	Global_LogPrint = true
}

func Disable_LogPrint() {
	Global_LogPrint = false
}

func Enable_InfoPrint() {
	Global_InfoPrint = true
}

func Disable_InfoPrint() {
	Global_InfoPrint = false
}

func InfoPrint(format string, a ...interface{}) (int, error) {
	if !Global_InfoPrint {
		return 0, nil
	}
	return fmt.Printf(format, a...)
}

func LogPrint(format string, a ...interface{}) (int, error) {
	if !Global_LogPrint {
		return 0, nil
	}
	_, file, _, ok := runtime.Caller(1)
	paths := strings.Split(file, "/")
	callerPackage := paths[len(paths)-2]
	if ok {
		format = fmt.Sprintf(White("[%s]: %s\n"), callerPackage, format)
		return fmt.Printf(format, a...)
	}
	return fmt.Printf(format+"\n", a...)
}
