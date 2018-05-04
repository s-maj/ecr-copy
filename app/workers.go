package app

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/awserr"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
)

func copyReposWorkers(svcDst *ecr.ECR, repoCh <-chan string, controlCh <-chan struct{}) {
	for {
		select {
		case r := <-repoCh:
			_, err := svcDst.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{
				RepositoryNames: []string{r},
			}).Send()
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() == "RepositoryNotFoundException" {
						_, err := svcDst.CreateRepositoryRequest(&ecr.CreateRepositoryInput{
							RepositoryName: aws.String(r),
						}).Send()
						if err != nil {
							log.Error(err.Error())
						}
					}
				} else {
					log.Error(err.Error())
				}
			}

		case <-controlCh:
			return
		}
	}
}

func imageWorkers(svcSrc, svcDst *ecr.ECR, imageCh <-chan ecr.ImageDetail, dockerCh chan<- map[string]string, controlCh <-chan struct{}) {
	for {
		select {
		case srcImage := <-imageCh:
			for _, tag := range srcImage.ImageTags {

				resp, err := svcDst.DescribeImagesRequest(&ecr.DescribeImagesInput{
					RepositoryName: srcImage.RepositoryName,
					ImageIds: []ecr.ImageIdentifier{
						ecr.ImageIdentifier{
							ImageTag: aws.String(tag),
						}},
				}).Send()

				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {

						// Handle scenario where tag does not exist
						if awsErr.Code() == "ImageNotFoundException" {
							scheduleImageCopy(svcSrc, svcDst, srcImage, tag, dockerCh)
						} else {
							log.Error(err.Error())
						}
					}

					// Image exists in destination repository
				} else {
					for _, dstImage := range resp.ImageDetails {
						if checkImageSha256(dstImage, srcImage) != true {
							scheduleImageCopy(svcSrc, svcDst, srcImage, tag, dockerCh)
						}
					}
				}
			}

		case <-controlCh:
			return
		}
	}
}

func dockerWorkers(svcSrc, svcDst *ecr.ECR, dockerCh <-chan map[string]string, controlCh <-chan struct{}) {
	// Spawn docker connector
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerApiVer))
	if err != nil {
		log.Println(err.Error())
	}

	// Obtain docker auth tokens
	srcToken := getDockerToken(svcSrc)
	dstToken := getDockerToken(svcDst)

	defer func() {
		if err := recover(); err != nil {
			go dockerWorkers(svcSrc, svcDst, dockerCh, controlCh)
		}
	}()

	for {
		select {
		case image := <-dockerCh:
			// Pull image
			log.Info(fmt.Sprintf("Pulling %s", image["src"]))
			out, err := cli.ImagePull(ctx, image["src"], types.ImagePullOptions{RegistryAuth: srcToken})
			if err != nil {
				log.Error(err.Error())

			}
			eventsHandler(out, image["src"])

			// Tag image to match destination repository
			err = cli.ImageTag(ctx, image["src"], image["dst"])
			if err != nil {
				log.Error(err.Error())
			}

			// Push image
			log.Info(fmt.Sprintf("Pushing %s", image["dst"]))
			out, err = cli.ImagePush(ctx, image["dst"], types.ImagePushOptions{RegistryAuth: dstToken})
			if err != nil {
				log.Error(err.Error())
			}
			eventsHandler(out, image["dst"])

		case <-controlCh:
			return
		}
	}
}
