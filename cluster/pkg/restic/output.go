package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
	oneliners "github.com/the-redback/go-oneliners"
)

type BackupOutput struct {
	Snapshot       string    `json:"snapshot,omitempty"`
	Size           string    `json:"size,omitempty"`
	Uploaded       string    `json:"uploaded,omitempty"`
	ProcessingTime string    `json:"processingTime,omitempty"`
	FileStats      FileStats `json:"fileStats,omitempty"`
}

type FileStats struct {
	TotalFiles      *int `json:"totalFiles,omitempty"`
	NewFiles        *int `json:"newFiles,omitempty"`
	ModifiedFiles   *int `json:"modifiedFiles,omitempty"`
	UnmodifiedFiles *int `json:"unmodifiedFiles,omitempty"`
}

func WriteOutput(output []byte, outputDir string) error {
	fmt.Println("=============output=========\n", string(output))
	out, err := parseOutput(output)
	if err != nil {
		return err
	}

	jsonOuput, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return writeOutputJson(jsonOuput, outputDir)
}

func parseOutput(output []byte) (*BackupOutput, error) {
	res := &BackupOutput{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if strings.HasPrefix(line, "Files:") {
			info := strings.FieldsFunc(line, separators)
			fmt.Println("len: ", len(info), "Slice: ", info)
			if len(info) < 7 {
				return nil, fmt.Errorf("failed to parse files statistics")
			}
			newFiles, err := strconv.Atoi(info[1])
			if err != nil {
				return nil, err
			}
			modifiedFiles, err := strconv.Atoi(info[3])
			if err != nil {
				return nil, err
			}
			unmodifiedFiles, err := strconv.Atoi(info[5])
			if err != nil {
				return nil, err
			}
			res.FileStats.NewFiles = types.IntP(newFiles)
			res.FileStats.ModifiedFiles = types.IntP(modifiedFiles)
			res.FileStats.UnmodifiedFiles = types.IntP(unmodifiedFiles)
		} else if strings.HasPrefix(line, "Added to the repo:") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 6 {
				return nil, fmt.Errorf("failed to parse upload statistics")
			}
			res.Uploaded = info[length-2] + " " + info[length-1]
		} else if strings.HasPrefix(line, "processed") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 7 {
				return nil, fmt.Errorf("failed to parse file processing statistics")
			}
			totalFiles, err := strconv.Atoi(info[1])
			if err != nil {
				return nil, err
			}
			res.FileStats.TotalFiles = types.IntP(totalFiles)
			res.Size = info[3] + " " + info[4]
			res.ProcessingTime = info[6]
		} else if strings.HasPrefix(line, "snapshot") && strings.HasSuffix(line, "saved") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 3 {
				return nil, fmt.Errorf("failed to parse snapshot statistics")
			}
			res.Snapshot = info[1]
		}
	}
	oneliners.PrettyJson(res, "Response")
	return res, nil
}

func separators(r rune) bool {
	return r == ' ' || r == '\t' || r == ','
}
func writeOutputJson(data []byte, dir string) error {

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fileName := filepath.Join(dir, "output.json")
	if err := ioutil.WriteFile(fileName, data, 0755); err != nil {
		return err
	}

	return nil
}
