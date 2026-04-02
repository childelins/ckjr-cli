package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/childelins/ckjr-cli/internal/cmdgen"
	"github.com/childelins/ckjr-cli/internal/logging"
	"github.com/childelins/ckjr-cli/internal/ossupload"
	"github.com/childelins/ckjr-cli/internal/output"
)

func newUploadImageCmd(clientFactory cmdgen.APIClientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "upload-image [json]",
		Short: "将外部图片URL转存到系统素材库",
		Long:  "下载外部图片链接，直传到 OSS 并保存到素材库。返回素材库中的图片 URL。",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var input map[string]interface{}
			if len(args) > 0 {
				if err := json.Unmarshal([]byte(args[0]), &input); err != nil {
					output.PrintError(os.Stderr, "JSON 解析失败: "+err.Error())
					os.Exit(1)
				}
			}

			imageURL, _ := input["url"].(string)
			if imageURL == "" {
				output.PrintError(os.Stderr, "缺少 url 参数")
				os.Exit(1)
			}

			client, err := clientFactory()
			if err != nil {
				output.PrintError(os.Stderr, err.Error())
				os.Exit(1)
			}

			pretty, _ := cmd.Flags().GetBool("pretty")
			verbose, _ := cmd.Flags().GetBool("verbose")

			ctx := logging.WithRequestID(context.Background(), logging.NewRequestID())

			result, err := ossupload.Upload(ctx, client, imageURL)
			if err != nil {
				if verbose {
					output.PrintError(os.Stderr, err.Error())
				} else {
					output.PrintError(os.Stderr, formatUploadError(err))
				}
				os.Exit(1)
			}

			output.Print(os.Stdout, result, pretty)
		},
	}
}

func formatUploadError(err error) string {
	return fmt.Sprintf("图片上传失败: %v", err)
}
