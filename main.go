package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

/*
TODO:
- Improve design
- Create optional GUI
- Handle errors
*/

type ResourceStats struct {
	count     int
	totalSize int64
}

type CodeStats struct {
	ResourceStats
	lineCount      int
	emptyLineCount int
	charCount      int
}

type SceneStats struct {
	ResourceStats
	subResourceCount int
	nodeCount        int
}

type File struct {
	name string
	size int64
}

var resourceStats ResourceStats
var scriptStats CodeStats
var shaderStats CodeStats
var sceneStats SceneStats
var largestFile File
var smallestFile File

func main() {
	gdProjectFilePath := os.Args[1]
	gdProjectDir := filepath.Dir(gdProjectFilePath)

	projectName, err := getProjectName(gdProjectFilePath)
	if err != nil {
		panic(err)
	}
	scanDirAndUpdateStats(gdProjectDir)

	fmt.Printf("Project Name: %s\n", projectName)
	fmt.Println()

	fmt.Println("Scene Stats:")
	sceneStats.print()
	fmt.Println()

	fmt.Println("Script Stats:")
	scriptStats.print()
	fmt.Println()

	fmt.Println("Shader Stats:")
	shaderStats.print()
	fmt.Println()

	fmt.Println("Resource Stats:")
	resourceStats.print()
	fmt.Println()

	fmt.Printf("Largest File: %s (%dkB)\n", largestFile.name, largestFile.size)
	fmt.Printf("Smallest File: %s (%dkN)\n", smallestFile.name, smallestFile.size)
	fmt.Println()
}

func scanDirAndUpdateStats(dir string) {
	if entries, err := os.ReadDir(dir); err == nil {

		for _, entry := range entries {
			if isHiddenFile(entry) {
				continue
			}

			if entry.IsDir() {
				if entry.Name() != "addons" {
					scanDirAndUpdateStats(dir + "/" + entry.Name())
				}
			} else {
				ext := filepath.Ext(entry.Name())[1:]
				fp := dir + "/" + entry.Name()

				file, err := os.Open(fp)

				if err != nil {
					panic(err)
				}

				switch ext {
				case "gdshader":
					shaderStats.update(file)
				case "gd", "cs":
					scriptStats.update(file)
				case "tscn":
					sceneStats.update(file)
				default:
					// A resource is a non-hidden file which is not a shader, script, or scene.
					resourceStats.update(file)
				}

				file.Close()
			}
		}
	} else {
		panic(err)
	}
}

func (stats *ResourceStats) update(file *os.File) {
	fileInfo, err := file.Stat()

	if err != nil {
		panic(err)
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	// This works for now but design wise I don't like it
	// I don't think a struct func should update a global var
	// As a side effect
	if fileSize > largestFile.size {
		largestFile.name = fileName
		largestFile.size = fileSize
	}

	if fileSize < smallestFile.size || smallestFile.size == 0 {
		smallestFile.name = fileName
		smallestFile.size = fileSize
	}
	//

	stats.totalSize += fileSize
	stats.count++
}

func (stats ResourceStats) print() {
	fmt.Printf("Count: %d\n", stats.count)
	fmt.Printf("Size: %dkB\n", stats.totalSize)
}

func (stats *CodeStats) update(file *os.File) {
	stats.ResourceStats.update(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if len(line) == 0 {
			stats.emptyLineCount++
		} else {
			stats.lineCount++
		}

		for _, char := range line {
			letter := string(char)

			if letter != "\n" {
				stats.charCount++
			}
		}
	}
}

func (stats CodeStats) print() {
	stats.ResourceStats.print()
	fmt.Printf("Line Count: %d\n", stats.lineCount)
	fmt.Printf("Empty Line Count: %d\n", stats.emptyLineCount)
	fmt.Printf("Characters: %d\n", stats.charCount)
}

func (stats *SceneStats) update(file *os.File) {
	stats.ResourceStats.update(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "node name=") {
			stats.nodeCount++
		} else if strings.Contains(line, "sub_resource type=") {
			stats.subResourceCount++
		}
	}
}

func (stats SceneStats) print() {
	stats.ResourceStats.print()
	fmt.Printf("Node Count: %d\n", stats.nodeCount)
	fmt.Printf("Sub Resource Count: %d\n", stats.subResourceCount)
}

func isHiddenFile(entry fs.DirEntry) bool {
	return string(entry.Name()[0]) == "."
}

func hasGdIgnore(dirEntires []fs.DirEntry) bool {
	for _, entry := range dirEntires {
		if entry.Name() == ".gdignore" {
			return true
		}
	}
	return false
}

func getProjectName(gdProjectFilePath string) (string, error) {
	projectName := ""

	if data, err := os.ReadFile(gdProjectFilePath); err == nil {
		content := string(data)

		startIndex := strings.Index(content, "config/name=") + len("config/name=\"")
		endIndex := startIndex + strings.Index(content[startIndex:], "\"")
		projectName = content[startIndex:endIndex]
	} else {
		return "", err
	}

	return projectName, nil
}
