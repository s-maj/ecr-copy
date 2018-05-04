package cmd

import (
	"ecrcopy/app"
	"github.com/spf13/cobra"
)

var dstRegion string
var srcRegion string
var repoWorkers int
var imageWorkers int
var dockerWorkers int
var deamon bool

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copy AWS ECR repos from one AWS region to another",

	Run: func(cmd *cobra.Command, args []string) {
		app.CopyEcr(srcRegion, dstRegion, deamon, imageWorkers, repoWorkers, dockerWorkers)
	},
}

func init() {
	copyCmd.Flags().StringVarP(&srcRegion, "src", "s", "", "Source AWS region (eg us-west-2)")
	copyCmd.Flags().StringVarP(&dstRegion, "dst", "d", "", "Destination AWS region (eg eu-west-2)")
	copyCmd.MarkFlagRequired("dst")
	copyCmd.MarkFlagRequired("src")
	copyCmd.Flags().BoolVarP(&deamon, "deamon", "", false, "Specify to run program indefinitely")

	copyCmd.Flags().IntVarP(&dockerWorkers, "docker-workers", "", 8, "Number of Docker workers for pull/pull operations")
	copyCmd.Flags().IntVarP(&imageWorkers, "image-workers", "", 16, "Number of AWS ECR image workers for pull/pull operations")
	copyCmd.Flags().IntVarP(&repoWorkers, "repo-workers", "", 4, "Number of AWS ECR repo workers for pull/pull operations")

	rootCmd.AddCommand(copyCmd)
}
