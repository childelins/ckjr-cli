package update

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/updater"
)

var currentVersion = "dev"

// SetVersion 设置当前版本号，由 cmd/root.go 调用
func SetVersion(v string) {
	currentVersion = v
}

// apiURL 可在测试中覆盖
var defaultAPIURL = "https://api.github.com/repos/childelins/ckjr-cli/releases/latest"

// NewCommand 创建 update 命令
func NewCommand() *cobra.Command {
	var apiURL string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "检查并更新 ckjr-cli 到最新版本",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, apiURL)
		},
	}

	cmd.Flags().StringVar(&apiURL, "api-url", defaultAPIURL, "GitHub API URL（用于测试）")
	_ = cmd.Flags().MarkHidden("api-url")

	return cmd
}

func runUpdate(cmd *cobra.Command, apiURL string) error {
	if currentVersion == "dev" {
		return fmt.Errorf("当前为开发版本 (dev)，请使用 install.sh 安装正式版本")
	}

	fmt.Fprintln(cmd.OutOrStdout(), "正在检查更新...")

	latestVersion, downloadURL, err := updater.CheckLatestVersion(apiURL)
	if err != nil {
		return err
	}

	cmp, err := updater.CompareVersions(currentVersion, latestVersion)
	if err != nil {
		return err
	}

	if cmp >= 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "已是最新版本 (%s)\n", currentVersion)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "发现新版本: %s -> %s\n", currentVersion, latestVersion)
	fmt.Fprintln(cmd.OutOrStdout(), "正在下载更新...")

	if err := updater.DownloadAndReplace(downloadURL, ""); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "更新成功！%s -> %s\n", currentVersion, latestVersion)
	return nil
}
