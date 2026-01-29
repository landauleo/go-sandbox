package main

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

//OsInput -> Считать с клавы -> Read
//OsInput -> Считать с клавы -> Read

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

func dirTree(out io.Writer, path string, files bool) error {
	print(out, files, path, "")
	return nil
}

func print(out io.Writer, files bool, path string, prefix string) {
	res, _ := os.Open(path)
	if res == nil {
		return
	}

	dirEntries, _ := res.ReadDir(0)

	//sort
	slices.SortFunc(dirEntries, func(a, b os.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	//filter
	if !files {
		dirEntries = slices.DeleteFunc(dirEntries, func(entry os.DirEntry) bool {
			return !entry.IsDir()
		})
	}

	for index, item := range dirEntries {

		isLast := len(dirEntries) == index+1
		var tab string
		if isLast {
			tab = "└───"
		} else {
			tab = "├───"
		}

		var result string
		if item.IsDir() {
			result = fmt.Sprintf("%s%s%s\n", prefix, tab, item.Name()) //тут префикс от родителя!!!
		} else {
			info, _ := item.Info()
			var size = info.Size()
			var sizeStr string
			if size == 0 {
				sizeStr = "empty"
			} else {
				sizeStr = fmt.Sprintf("%db", info.Size())
			}
			result = fmt.Sprintf("%s%s%s (%s)\n", prefix, tab, item.Name(), sizeStr) //тут префикс от родителя!!!
		}

		out.Write([]byte(result))

		newPrefix := prefix //тут бцдцщий префикс для ребёнка (именно он и передается в рекурсивный метод!!!)
		if isLast {
			newPrefix += "\t"
		} else {
			newPrefix += "│\t"
		}

		print(out, files, path+"/"+item.Name(), newPrefix)
	}

}
