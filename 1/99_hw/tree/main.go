package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

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
	res, err := os.Open(path)
	if err != nil {
		fmt.Println("open failed: %w", err)
	}
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
			size := info.Size()
			sizeStr := "empty"
			if size != 0 {
				sizeStr = fmt.Sprintf("%db", info.Size())
			}
			result = fmt.Sprintf("%s%s%s (%s)\n", prefix, tab, item.Name(), sizeStr) //тут префикс от родителя!!!
		}

		out.Write([]byte(result))

		newPrefix := prefix //тут будущий префикс для ребёнка (именно он и передается в рекурсивный метод!!!)
		if isLast {
			newPrefix += "\t"
		} else {
			newPrefix += "│\t"
		}

		res.Close()
		print(out, files, filepath.Join(path, item.Name()), newPrefix)
	}

}
