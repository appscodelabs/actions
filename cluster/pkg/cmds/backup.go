package cmds

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/appscode/go/types"

	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/appscode/kutil/tools/backup"
	"github.com/appscode/kutil/tools/clientcmd"
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus/push"

	"github.com/appscode/go/flags"
	"github.com/appscodelabs/actions/cluster/pkg/restic"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type options struct {
	kubeconfigPath string
	context        string
	scratchDir     string
	sanitize       bool
	backupDir      string
	provider       string
	bucket         string
	endpoint       string
	path           string
	secretDir      string
	enableCache    bool
	hostname       string
	outputDir      string

	pushgatewayURL  string
	retentionPolicy retentionPolicy
}

type retentionPolicy struct {
	policy string
	value  string
	prune  bool
	dryRun bool
}

func NewCmdBackup() *cobra.Command {

	opt := options{
		backupDir:   "/tmp/restic/backup",
		scratchDir:  "/tmp/restic/scratch",
		enableCache: false,
		outputDir:   "/tmp/restic/output",
	}

	cmd := &cobra.Command{
		Use:               "backup",
		Short:             "Takes a backup of Kubernetes api objects",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "kubeconfig", "context", "provider", "path", "secret-dir")

			// Dump YAML of cluster resources
			restConfig, err := clientcmd.BuildConfigFromContext(opt.kubeconfigPath, opt.context)
			if err != nil {
				return err
			}
			mgr := backup.NewBackupManager(opt.context, restConfig, opt.sanitize)
			filename, err := mgr.BackupToDir(opt.backupDir)
			if err != nil {
				return err
			}
			fmt.Printf("Cluster objects are stored in %s", filename)
			fmt.Println()

			// Setup Environment variables for restic cli
			w := restic.New(opt.scratchDir, opt.enableCache, opt.hostname)
			err = w.SetupEnv(opt.provider, opt.bucket, opt.endpoint, opt.path, opt.secretDir)
			if err != nil {
				return err
			}

			// Initialize restic repository if it does not exist
			_, err = w.InitRepositoryIfAbsent()
			if err != nil {
				return err
			}

			// Backup YAML of cluster resources that has been dumped in the directory pointed by opt.backupDir
			out, err := w.Backup(opt.backupDir, nil)
			if err != nil {
				fmt.Println(err)
				return err
			}

			// Parse output of backup command
			backupOutput, err := restic.ParseBackupOutput(out)
			if err != nil {
				return err
			}

			// Check repository integrity
			out, err = w.Check()
			if err != nil {
				return err
			}
			// Parse output of "check" command
			backupOutput.Integrity = types.BoolP(restic.ParseCheckOutput(out))

			// Cleanup old snapshot according to retention policy
			out, err = w.Cleanup(opt.retentionPolicy.policy, opt.retentionPolicy.value, opt.retentionPolicy.prune, opt.retentionPolicy.dryRun)
			if err != nil {
				return err
			}
			// Parse output of cleanup command to extract information
			fmt.Println(string(out))


			// Read repository statics after cleanup
			out,err=w.Stats()
			if err!=nil{
				return err
			}
			fmt.Println("==================================================\n",string(out))
			// Write output of "backup" command into output.json to the directory pointed by opt.outputDir
			err = restic.WriteOutput(backupOutput, opt.outputDir)
			if err != nil {
				return err
			}

			// Generate Prometheus metrics from backupOutput
			backupMetrics := restic.NewBackupMetrics()
			err = backupMetrics.SetValues(backupOutput)
			if err != nil {
				return err
			}

			// Write Metrics to metrics.prom file in output directory
			registry := prometheus.NewRegistry()
			registry.MustRegister(
				backupMetrics.FileMetrics.TotalFiles,
				backupMetrics.FileMetrics.NewFiles,
				backupMetrics.FileMetrics.ModifiedFiles,
				backupMetrics.FileMetrics.UnmodifiedFiles,
				backupMetrics.DataSize,
				backupMetrics.DataUploaded,
				backupMetrics.DataProcessingTime,
				backupMetrics.RepoIntegrity,
			)

			// If pushgatewayURL is provided then push metrics to the Pushgateway otherwise write into a file in output directory
			if opt.pushgatewayURL != "" {
				pusher := push.New(opt.pushgatewayURL, "cluster-backup")
				err = pusher.Gatherer(registry).Push()
				if err != nil {
					return nil
				}
			}
			err = prometheus.WriteToTextfile(filepath.Join(opt.outputDir, "metrics.prom"), registry)
			if err != nil {
				return err
			}
			_, err = ioutil.ReadFile(filepath.Join(opt.outputDir, "metrics.prom"))
			if err != nil {
				return err
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&opt.sanitize, "sanitize", false, " Sanitize fields in YAML")
	cmd.Flags().StringVar(&opt.backupDir, "backup-dr", opt.backupDir, "Directory where YAML files will be stored")
	cmd.Flags().StringVar(&opt.kubeconfigPath, "kubeconfig", "", "kubeconfig file pointing at the 'core' kubernetes server")
	cmd.Flags().StringVar(&opt.context, "context", "", "Name of the kubeconfig context to use")
	cmd.Flags().StringVar(&opt.provider, "provider", "", "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.bucket, "bucket", "", "bucket name")
	cmd.Flags().StringVar(&opt.endpoint, "endpoint", "", "endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.path, "path", "", "directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&opt.secretDir, "secret-dir", "", "directory where storage secret has been mounted")
	cmd.Flags().BoolVar(&opt.enableCache, "cache", opt.enableCache, "weather to enable cache")
	cmd.Flags().StringVar(&opt.hostname, "hostname", "", "name of the host machine")
	cmd.Flags().StringVar(&opt.outputDir, "output-dir", opt.outputDir, "Directory where output.json file will be written")
	cmd.Flags().StringVar(&opt.pushgatewayURL, "pushgateway-url", "", "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&opt.retentionPolicy.policy, "retention-policy.policy", "", "")
	cmd.Flags().StringVar(&opt.retentionPolicy.value, "retention-policy.value", "", "")
	cmd.Flags().BoolVar(&opt.retentionPolicy.prune, "retention-policy.prune", false, "")
	cmd.Flags().BoolVar(&opt.retentionPolicy.dryRun, "retention-policy.dryrun", false, "")
	return cmd
}
