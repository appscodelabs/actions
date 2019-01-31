package restic

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/appscode/go/log"
	"github.com/pkg/errors"
)

const (
	Exe = "/usr/local/bin/restic"
)

const (
	KeepLast    string = "--keep-last"
	KeepHourly  string = "--keep-hourly"
	KeepDaily   string = "--keep-daily"
	KeepWeekly  string = "--keep-weekly"
	KeepMonthly string = "--keep-monthly"
	KeepYearly  string = "--keep-yearly"
	KeepTag     string = "--keep-tag"
)

type Snapshot struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Tree     string    `json:"tree"`
	Paths    []string  `json:"paths"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	UID      int       `json:"uid"`
	Gid      int       `json:"gid"`
	Tags     []string  `json:"tags"`
}

func (w *ResticWrapper) ListSnapshots(snapshotIDs []string) ([]Snapshot, error) {
	result := make([]Snapshot, 0)
	args := w.appendCacheDirFlag([]interface{}{"snapshots", "--json", "--quiet", "--no-lock"})
	args = w.appendCaCertFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}

	err := w.sh.Command(Exe, args...).UnmarshalJSON(&result)
	return result, err
}

func (w *ResticWrapper) DeleteSnapshots(snapshotIDs []string) ([]byte, error) {
	args := w.appendCacheDirFlag([]interface{}{"forget", "--quiet", "--prune"})
	args = w.appendCaCertFlag(args)
	for _, id := range snapshotIDs {
		args = append(args, id)
	}

	return w.run(Exe, args)
}

func (w *ResticWrapper) InitRepositoryIfAbsent() ([]byte, error) {
	args := w.appendCacheDirFlag([]interface{}{"snapshots", "--json"})
	args = w.appendCaCertFlag(args)
	if _, err := w.run(Exe, args); err != nil {
		args = w.appendCacheDirFlag([]interface{}{"init"})
		args = w.appendCaCertFlag(args)

		return w.run(Exe, args)
	}
	return nil, nil
}

func (w *ResticWrapper) Backup(path string, tags []string) ([]byte, error) {
	args := []interface{}{"backup", path}
	if w.hostname != "" {
		args = append(args, "--host")
		args = append(args, w.hostname)
	}
	// add tags if any
	for _, tag := range tags {
		args = append(args, "--tag")
		args = append(args, tag)
	}
	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) Cleanup(policy, value string, prune, dryRun bool) ([]byte, error) {

	args := []interface{}{"forget"}

	args = append(args, policy)
	args = append(args, value)

	if prune {
		args = append(args, "--prune")
	}

	if dryRun {
		args = append(args, "--dry-run")
	}

	if len(args) > 1 {
		args = w.appendCacheDirFlag(args)
		args = w.appendCaCertFlag(args)

		return w.run(Exe, args)
	}
	return nil, nil
}

func (w *ResticWrapper) Restore(path, host, snapshotID string) ([]byte, error) {
	args := []interface{}{"restore"}
	if snapshotID != "" {
		args = append(args, snapshotID)
	} else {
		args = append(args, "latest")
	}
	args = append(args, "--path")
	args = append(args, path) // source-path specified in restic fileGroup
	args = append(args, "--host")
	args = append(args, host)

	// Remove last part from the path.
	// https://github.com/appscode/stash/issues/392
	args = append(args, "--target")
	args = append(args, filepath.Dir(path))

	args = w.appendCacheDirFlag(args)
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) Check() ([]byte, error) {
	args := w.appendCacheDirFlag([]interface{}{"check"})
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) Stats() ([]byte, error) {
	args := w.appendCacheDirFlag([]interface{}{"stats"})
	args = append(args,"--mode=raw-data","--quiet")
	args = w.appendCaCertFlag(args)

	return w.run(Exe, args)
}

func (w *ResticWrapper) appendCacheDirFlag(args []interface{}) []interface{} {
	if w.enableCache {
		cacheDir := filepath.Join(w.scratchDir, "restic-cache")
		return append(args, "--cache-dir", cacheDir)
	}
	return append(args, "--no-cache")
}

func (w *ResticWrapper) appendCaCertFlag(args []interface{}) []interface{} {
	if w.cacertFile != "" {
		return append(args, "--cacert", w.cacertFile)
	}
	return args
}

func (w *ResticWrapper) run(cmd string, args []interface{}) ([]byte, error) {
	out, err := w.sh.Command(cmd, args...).Output()
	if err != nil {
		log.Errorf("Error running command '%s %s' output:\n%s", cmd, args, string(out))
		parts := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
		if len(parts) > 1 {
			parts = parts[len(parts)-1:]
			return nil, errors.New(parts[0])
		}
	}
	return out, err
}
