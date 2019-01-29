package cmds

import (
	"fmt"

	"github.com/appscode/kutil/tools/backup"
	"github.com/appscode/kutil/tools/clientcmd"

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

	outputDir string
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

			// ============ Dump YAML of cluster resources =======================
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

			// ============= Upload the dumped YAML using restic ====================
			w := restic.New(opt.scratchDir, opt.enableCache, opt.hostname)
			err = w.SetupEnv(opt.provider, opt.bucket, opt.endpoint, opt.path, opt.secretDir)
			if err != nil {
				fmt.Println(err)
				return err
			}
			_, err = w.InitRepositoryIfAbsent()
			if err != nil {
				fmt.Println(err)
				return err
			}

			out, err := w.Backup(opt.backupDir, nil)
			if err != nil {
				fmt.Println(err)
				return err
			}
			// ================== Write Output of backup command into output.json =============
			err = restic.WriteOutput(out, opt.outputDir)
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
	return cmd
}
