package extern

import (
	"github.com/kgwinnup/go-yara/yara"
	"io/ioutil"
	"os"
	"syscall"
)

func IsExecutable(fname string) bool {
	err := syscall.Access(fname, X_OK)
	return err == nil
}
func IsReadable(fname string) bool {
	err := syscall.Access(fname, R_OK)
	return err == nil
}
func CurrDir() string {
	currdir, _ := os.Getwd()
	return currdir
}
func YaraRule(pathrule string) (yr *yara.Yara, err error) {
	rule, err := ioutil.ReadFile(pathrule)
	if err != nil {
		return nil, err
	}
	yr, err = yara.New(string(rule))
	if err != nil {
		return nil, err
	}
	return yr, nil
}
