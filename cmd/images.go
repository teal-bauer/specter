package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/teal-bauer/specter/api"
	"github.com/teal-bauer/specter/internal/config"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage images",
}

var imagesUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload an image",
	Args:  cobra.ExactArgs(1),
	RunE:  runImagesUpload,
}

var imageRef string

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.AddCommand(imagesUploadCmd)

	imagesUploadCmd.Flags().StringVar(&imageRef, "ref", "", "Reference name for the image")
}

func runImagesUpload(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client := api.NewClient(cfg)

	url, err := client.UploadImage(args[0], imageRef)
	if err != nil {
		return err
	}

	if config.OutputFormat() == "json" {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{
			"url": url,
			"ref": imageRef,
		})
	}

	fmt.Println(url)
	return nil
}
