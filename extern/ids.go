package extern

import (
	"bufio"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Either it is configured in snort or these two constants need changing
// All the files within DirSnort will be matched against the pattern to see if
// they need to be looked at.
const DirSnort = "/var/log/snort"
const FilesToPayAttention = `^.*/rips\.`

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false // os.IsNotExist(err)
	}
	if fi.IsDir() {
		return true
	}
	return false
}

func isFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false //os.IsNotExist(err)
	}
	if fi.Mode().IsRegular() {
		return true
	}
	return false
}

func FgrepAll(paths map[string]bool, needle string) bool {
	for path := range paths {
		ispres := Fgrep(path, needle)
		if ispres {
			return true
		}
	}
	return false
}

func Fgrep(path string, needle string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	br := bufio.NewReader(file)

	for {
		ln, err := br.ReadString('\n')
		if err != nil {
			return false
		}
		if strings.Contains(ln, needle) {
			return true
		}
	}
}

func Watcher(dirpath string, pathsc chan string, exitc chan int) (err error) {
	re := regexp.MustCompile(FilesToPayAttention)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	err = watcher.Add(dirpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot walk path to ids logs %s\n", dirpath)
		return err
	}

	walkfunc := func(path string, fi os.FileInfo, err error) error {
		if err == nil && fi.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot walk path to ids logs %s\n", dirpath)
				return err
			}
		}
		return err
	}

	err = filepath.Walk(dirpath, walkfunc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot walk path to ids logs %s\n", dirpath)
		return err
	}
	go func() {
		defer watcher.Close()
		for {
			select {
			case <-exitc:
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) {
					if isDir(event.Name) {
						watcher.Add(event.Name)
					}
				}
				if event.Has(fsnotify.Write) {
					if isFile(event.Name) && re.MatchString(event.Name) {
						pathsc <- event.Name
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	return nil
}
