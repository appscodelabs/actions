package cmds

import (
	"github.com/appscode/go/flags"
	"github.com/appscode/kutil/tools/backup"
	"github.com/appscode/kutil/tools/restic"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd"
)

type options struct {
	masterUrl      string
	kubeconfigPath string
	context        string
	sanitize       bool
	backupDir      string
	backup         restic.BackupOptions
	metrics        restic.MetricsOptions
}

const (
	JobClusterTools = "cluster-tool"
)

func NewCmdBackup() *cobra.Command {

	opt := options{
		backupDir: "/tmp/restic/backup",
		backup: restic.BackupOptions{
			ScratchDir:  "/tmp/restic/scratch",
			EnableCache: false,
		},
	}

	cmd := &cobra.Command{
		Use:               "backup",
		Short:             "Takes a backup YAMLs of Kubernetes api objects",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.EnsureRequiredFlags(cmd, "provider", "path", "secret-dir", "retention-policy.policy", "retention-policy.value")

			// Run backup
			backupOutput, backupErr := runBackup(&opt.backup, opt.masterUrl, opt.kubeconfigPath, opt.context, opt.backupDir, opt.sanitize)

			// If metrics are enabled then generate metrics
			if opt.metrics.Enabled {
				err := opt.metrics.HandleMetrics(backupOutput, backupErr, JobClusterTools)
				if err != nil {
					return errors.NewAggregate([]error{backupErr, err})
				}
			}

			// If output directory specified, then write the output in "output.json" file in the specified directory
			if backupErr == nil && opt.backup.OutputDir != "" {
				err := restic.WriteOutput(backupOutput, opt.backup.OutputDir)
				if err != nil {
					return err
				}
			}
			return backupErr
		},
	}
	cmd.Flags().StringVar(&opt.masterUrl, "master-url", "", "URL of master node")
	cmd.Flags().StringVar(&opt.kubeconfigPath, "kubeconfig", opt.kubeconfigPath, "kubeconfig file pointing at the 'core' kubernetes server")
	cmd.Flags().StringVar(&opt.context, "context", "", "Context to use from kubeconfig file")
	cmd.Flags().BoolVar(&opt.sanitize, "sanitize", false, " Sanitize YAML files")
	cmd.Flags().StringVar(&opt.backupDir, "backup-dir", opt.backupDir, "Directory where dumped YAML files will be stored temporarily")

	cmd.Flags().BoolVar(&opt.backup.EnableCache, "cache", opt.backup.EnableCache, "Specify weather to enable caching for restic")
	cmd.Flags().StringVar(&opt.backup.Hostname, "hostname", "", "Name of the host machine")

	cmd.Flags().StringVar(&opt.backup.Provider, "provider", "", "Backend provider (i.e. gcs, s3, azure etc)")
	cmd.Flags().StringVar(&opt.backup.SecretDir, "secret-dir", "", "Directory where storage secret has been mounted")
	cmd.Flags().StringVar(&opt.backup.Bucket, "bucket", "", "Name of the cloud bucket/container (keep empty for local backend)")
	cmd.Flags().StringVar(&opt.backup.Endpoint, "endpoint", "", "Endpoint for s3/s3 compatible backend")
	cmd.Flags().StringVar(&opt.backup.Path, "path", "", "Directory inside the bucket where backup will be stored")
	cmd.Flags().StringVar(&opt.backup.OutputDir, "output-dir", "", "Directory where output.json file will be written (keep empty if you don't need to write output in file)")

	cmd.Flags().StringVar(&opt.backup.RetentionPolicy.Policy, "retention-policy.policy", "", "Specify a retention policy")
	cmd.Flags().StringVar(&opt.backup.RetentionPolicy.Value, "retention-policy.value", "", "Value for specified retention policy")
	cmd.Flags().BoolVar(&opt.backup.RetentionPolicy.Prune, "retention-policy.prune", false, "Specify weather to prune old snapshot data")
	cmd.Flags().BoolVar(&opt.backup.RetentionPolicy.DryRun, "retention-policy.dryrun", false, "Specify weather to test retention policy without deleting actual data")

	cmd.Flags().BoolVar(&opt.metrics.Enabled, "metrics.enabled", false, "Specify weather to export Prometheus metrics")
	cmd.Flags().StringVar(&opt.metrics.PushgatewayURL, "metrics.pushgateway-url", "", "Pushgateway URL where the metrics will be pushed")
	cmd.Flags().StringVar(&opt.metrics.MetricFileDir, "metrics.dir", "", "Directory where to write metric.prom file (keep empty if you don't want to write metric in a text file)")
	cmd.Flags().StringSliceVar(&opt.metrics.Labels, "metrics.labels", nil, "Labels to apply in exported metrics")

	return cmd
}

func runBackup(backupOpt *restic.BackupOptions, masterUrl, kubeconfigPath, context, backupDir string, sanitize bool) (*restic.BackupOutput, error) {
	config, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfigPath)
	if err != nil {
		return nil, err
	}

	if context == "" {
		cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
		if err == nil {
			context = cfg.CurrentContext
		}else{
			// using incluster config. so no context. use default.
			context = "default"
		}
	}
	mgr := backup.NewBackupManager(context, config, sanitize)

	_, err = mgr.BackupToDir(backupDir)
	if err != nil {
		return nil, err
	}

	// Setup Environment variables for restic cli
	w := restic.NewResticWrapper(backupOpt.ScratchDir, backupOpt.EnableCache, backupOpt.Hostname)
	err = w.SetupEnv(backupOpt.Provider, backupOpt.Bucket, backupOpt.Endpoint, backupOpt.Path, backupOpt.SecretDir)
	if err != nil {
		return nil, err
	}

	// Initialize restic repository if it does not exist
	_, err = w.InitRepositoryIfAbsent()
	if err != nil {
		return nil, err
	}

	// Backup the dumped YAMLs stored temporarily in opt.backupDir
	out, err := w.Backup(backupDir, nil)
	if err != nil {
		return nil, err
	}

	backupOutput := &restic.BackupOutput{}

	// Extract information from the output of backup command
	err = backupOutput.ExtractBackupInfo(out)
	if err != nil {
		return nil, err
	}

	// Check repository integrity
	out, err = w.Check()
	if err != nil {
		return nil, err
	}
	// Extract information from output of "check" command
	backupOutput.ExtractCheckInfo(out)

	// Cleanup old snapshot according to retention policy
	out, err = w.Cleanup(backupOpt.RetentionPolicy.Policy, backupOpt.RetentionPolicy.Value, backupOpt.RetentionPolicy.Prune, backupOpt.RetentionPolicy.DryRun)
	if err != nil {
		return nil, err
	}
	// Extract information from output of cleanup command
	err = backupOutput.ExtractCleanupInfo(out)
	if err != nil {
		return nil, err
	}

	// Read repository statics after cleanup
	out, err = w.Stats()
	if err != nil {
		return nil, err
	}
	// Extract information from output of "stats" command
	err = backupOutput.ExtractStatsInfo(out)
	if err != nil {
		return nil, err
	}
	return backupOutput, nil
}
