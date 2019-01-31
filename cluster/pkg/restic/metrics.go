package restic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/prometheus/client_golang/prometheus"
)

type BackupMetrics struct {
	// DataSize shows total size of the target data to backup (in bytes)
	DataSize prometheus.Gauge
	// DataUploaded shows the amount of data uploaded to the repository in this session (in bytes)
	DataUploaded prometheus.Gauge
	// DataProcessingTime shows total time taken to backup the target data
	DataProcessingTime prometheus.Gauge
	// FileMetrics shows information of backup files
	FileMetrics *FileMetrics
	// RepoIntegrity shows result of repository integrity check after last backup
	RepoIntegrity prometheus.Gauge
}

type FileMetrics struct {
	// TotalFiles shows total number of files that has been backed up
	TotalFiles prometheus.Gauge
	// NewFiles shows total number of new files that has been created since last backup
	NewFiles prometheus.Gauge
	// ModifiedFiles shows total number of files that has been modified since last backup
	ModifiedFiles prometheus.Gauge
	// UnmodifiedFiles shows total number of files that has not been changed since last backup
	UnmodifiedFiles prometheus.Gauge
}

func NewBackupMetrics() *BackupMetrics {

	return &BackupMetrics{
		DataSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "restic",
				Subsystem:   "backup",
				Name:        "data_size_bytes",
				Help:        "Total size of the target data to backup (in bytes)",
				ConstLabels: nil,
			},
		),
		DataUploaded: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "restic",
				Subsystem:   "backup",
				Name:        "data_uploaded_bytes",
				Help:        "Amount of data uploaded to the repository in this session (in bytes)",
				ConstLabels: nil,
			},
		),
		DataProcessingTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "restic",
				Subsystem:   "backup",
				Name:        "data_processing_time_seconds",
				Help:        "Total time taken to backup the target data",
				ConstLabels: nil,
			},
		),
		FileMetrics: &FileMetrics{
			TotalFiles: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "total_files",
					Help:        "Total number of files that has been backed up",
					ConstLabels: nil,
				},
			),
			NewFiles: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "new_files",
					Help:        "Total number of new files that has been created since last backup",
					ConstLabels: nil,
				},
			),
			ModifiedFiles: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "modified_files",
					Help:        "Total number of files that has been modified since last backup",
					ConstLabels: nil,
				},
			),
			UnmodifiedFiles: prometheus.NewGauge(
				prometheus.GaugeOpts{
					Namespace:   "restic",
					Subsystem:   "backup",
					Name:        "unmodified_files",
					Help:        "Total number of files that has not been changed since last backup",
					ConstLabels: nil,
				},
			),
		},
		RepoIntegrity: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   "restic",
				Subsystem:   "repository",
				Name:        "integrity",
				Help:        "Result of repository integrity check after last backup",
				ConstLabels: nil,
			},
		),
	}
}

func (backupMetrics *BackupMetrics) SetValues(backupOutput *BackupOutput) error {
	dataSizeBytes, err := convertSizeToBytes(backupOutput.Size)
	if err != nil {
		return err
	}
	backupMetrics.DataSize.Set(dataSizeBytes)

	uploadSizeBytes, err := convertSizeToBytes(backupOutput.Uploaded)
	if err != nil {
		return err
	}
	backupMetrics.DataUploaded.Set(uploadSizeBytes)

	processingTimeSeconds, err := convertTimeToSeconds(backupOutput.ProcessingTime)
	if err != nil {
		return err
	}
	backupMetrics.DataProcessingTime.Set(float64(processingTimeSeconds))

	backupMetrics.FileMetrics.TotalFiles.Set(float64(*backupOutput.FileStats.TotalFiles))
	backupMetrics.FileMetrics.NewFiles.Set(float64(*backupOutput.FileStats.NewFiles))
	backupMetrics.FileMetrics.ModifiedFiles.Set(float64(*backupOutput.FileStats.ModifiedFiles))
	backupMetrics.FileMetrics.UnmodifiedFiles.Set(float64(*backupOutput.FileStats.UnmodifiedFiles))

	if *backupOutput.Integrity {
		backupMetrics.RepoIntegrity.Set(1)
	} else {
		backupMetrics.RepoIntegrity.Set(0)
	}
	return nil
}

func convertSizeToBytes(dataSize string) (float64, error) {
	parts := strings.Split(dataSize, " ")
	if len(parts) != 2 {
		return 0, errors.New("invalid data size format")
	}

	switch parts[1] {
	case "B":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size, nil
	case "KiB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024, nil
	case "MiB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024 * 1024, nil
	case "GiB":
		size, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, err
		}
		return size * 1024 * 1024 * 1024, nil
	}
	return 0, errors.New("unknown unit for data size")
}

func convertTimeToSeconds(processingTime string) (int, error) {
	var minutes, seconds int
	_, err := fmt.Sscanf(processingTime, "%dm%ds", &minutes, &seconds)
	if err != nil {
		return 0, err
	}

	return minutes*60 + seconds, nil
}
