package util

import "fmt"

func CheckError(err error, msg string, args ...interface{}) {
	if err != nil {
		ThrowError(msg+": %s", args, err)
	}
}

func ThrowError(msg string, args ...interface{}) {
	panic(fmt.Errorf(msg, args...))
}

var Global_LogPrint bool = false

func Enable_LogPrint() {
	Global_LogPrint = true
}

func LogPrint(format string, a ...interface{}) (int, error) {
	if !Global_LogPrint {
		return 0, nil
	}
	return fmt.Printf(format, a...)
}
