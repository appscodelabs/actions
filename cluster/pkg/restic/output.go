package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/the-redback/go-oneliners"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/appscode/go/types"
)

type BackupOutput struct {
	Snapshot       string    `json:"snapshot,omitempty"`
	Size           string    `json:"size,omitempty"`
	Uploaded       string    `json:"uploaded,omitempty"`
	ProcessingTime string    `json:"processingTime,omitempty"`
	FileStats      FileStats `json:"fileStats,omitempty"`
	Integrity      *bool     `json:"integrity,omitempty"`
}

type FileStats struct {
	TotalFiles      *int `json:"totalFiles,omitempty"`
	NewFiles        *int `json:"newFiles,omitempty"`
	ModifiedFiles   *int `json:"modifiedFiles,omitempty"`
	UnmodifiedFiles *int `json:"unmodifiedFiles,omitempty"`
}

func WriteOutput(out *BackupOutput, outputDir string) error {
	oneliners.PrettyJson(out, "Output")
	jsonOuput, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	return writeOutputJson(jsonOuput, outputDir)
}

func ParseBackupOutput(output []byte) (*BackupOutput, error) {
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
			m, s, err := convertToMinutesSeconds(info[6])
			if err != nil {
				return nil, err
			}
			res.ProcessingTime = fmt.Sprintf("%dm%ds", m, s)
		} else if strings.HasPrefix(line, "snapshot") && strings.HasSuffix(line, "saved") {
			info := strings.FieldsFunc(line, separators)
			length := len(info)
			if length < 3 {
				return nil, fmt.Errorf("failed to parse snapshot statistics")
			}
			res.Snapshot = info[1]
		}
	}
	return res, nil
}

func ParseCheckOutput(out []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		line = strings.TrimSpace(line)
		if line == "no errors were found" {
			return true
		}
	}
	return false
}

func convertToMinutesSeconds(time string) (int, int, error) {
	parts := strings.Split(time, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("failed to convert minutes")
	}
	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	fraction, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	seconds := int((fraction * 60) / 100)
	if seconds >= 60 {
		m := int(seconds / 60)
		minutes = minutes + m
		seconds = seconds - m*60
	}

	return minutes, seconds, nil
}

func separators(r rune) bool {
	return r == ' ' || r == '\t' || r == ','
}
func writeOutputJson(data []byte, dir string) error {

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Println("Failed to make directory: ", dir)
		return err
	}
	fileName := filepath.Join(dir, "output.json")
	if err := ioutil.WriteFile(fileName, data, 0755); err != nil {
		return err
	}

	return nil
}
