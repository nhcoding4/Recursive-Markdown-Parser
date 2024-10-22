package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ----------------------------------------------------------------------------------------------------------------
// Used to group filenames and raw data together
// ----------------------------------------------------------------------------------------------------------------

type FileData struct {
	rawData  string
	fileName string
}

func newFileData(rawData, fileName *string) *FileData {
	return &FileData{rawData: *rawData, fileName: *fileName}
}

// ----------------------------------------------------------------------------------------------------------------
// Load and save files.
// ----------------------------------------------------------------------------------------------------------------

type Files struct {
	paths          []string
	fileData       []*FileData
	saveFolderPath string
}

// ----------------------------------------------------------------------------------------------------------------
// Object creation
// ----------------------------------------------------------------------------------------------------------------

func NewFiles(paths []string) *Files {
	files := &Files{
		paths: paths,
	}
	files.createFolderPath()
	files.readFiles()

	return files
}

// ----------------------------------------------------------------------------------------------------------------
// Methods
// ----------------------------------------------------------------------------------------------------------------

func (f *Files) createFolder() {
	_, err := os.Stat(f.saveFolderPath)
	if os.IsNotExist(err) {
		err := os.Mkdir(f.saveFolderPath, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) createFolderPath() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(fmt.Errorf("unable to create filepath: %v", err))
		os.Exit(1)
	}

	folderLocation := "html_files"
	f.saveFolderPath = filepath.Join(wd, folderLocation)
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) createFiles() {
	for _, file := range *f.rawData() {
		block := NewBlocks(&file.rawData)

		parsedBlocks := make([]string, 0)

		for _, block := range *block.getBlocks() {
			parser := NewParser(&block)
			parsedBlocks = append(parsedBlocks, parser.parse().toHtml())
		}

		html := f.createHtml(&parsedBlocks)
		f.saveData(&html, &file.fileName)
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) createHtml(parsedBlocks *[]string) string {
	var out bytes.Buffer

	for _, block := range *parsedBlocks {
		out.WriteString(block + "\n")
	}

	return out.String()
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) createFilePath(fileName *string) string {
	name := strings.Split(*fileName, ".")[0]
	return filepath.Join(f.saveFolderPath, fmt.Sprintf("%v.html", name))
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) getTitle(parsedData *string) string {
	if strings.Contains(*parsedData, "h1") {
		return strings.Split(strings.Split(*parsedData, "<h1>")[1], "</h1>")[0]
	}
	return "Page"
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) rawData() *[]*FileData {
	return &f.fileData
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) readFiles() {
	for _, path := range f.paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Println(fmt.Errorf("error reading file from %v:  %v", path, err))
			continue
		}

		rawData := string(data[:])
		f.fileData = append(f.fileData, newFileData(&rawData, &path))
	}
}

// ----------------------------------------------------------------------------------------------------------------

func (f *Files) saveData(parsedData *string, filename *string) {
	f.createFolder()

	saveFile := f.boilerPlate()
	saveFile = strings.Replace(saveFile, "{{ Title }}", f.getTitle(parsedData), 1)
	saveFile = strings.Replace(saveFile, "{{ Content }}", *parsedData, 1)

	err := os.WriteFile(f.createFilePath(filename), []byte(saveFile), 0777)
	if err != nil {
		fmt.Println(fmt.Errorf("error creating file %v : %v", *filename, err))
	}
}

// ----------------------------------------------------------------------------------------------------------------
// Base for every file produced by the program
// ----------------------------------------------------------------------------------------------------------------

func (f *Files) boilerPlate() string {
	return `<!DOCTYPE html>
<html>

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title> {{ Title }} </title>
    <link href="./index.css" rel="stylesheet">
</head>

<body>
    <article>
        {{ Content }}
    </article>
</body>

</html>
`
}

// ----------------------------------------------------------------------------------------------------------------
