package app

import (
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"log"
	"os"
)

func CopyEcr(srcRegion, dstRegion string, deamon bool, imageWorkersNo, reposWorkersNo, dockerWorkersNo int) {
	for {
		cfg, err := external.LoadDefaultAWSConfig()
		if err != nil {
			log.Println("failed to get credentials, ", err)
			os.Exit(1)
		}

		// Get AWS connectors
		cfg.Region = srcRegion
		svcSrc := ecr.New(cfg)
		cfg.Region = dstRegion
		svcDst := ecr.New(cfg)

		// Clean images periodically
		go imageCleaner()

		copyRepos(svcSrc, svcDst, reposWorkersNo)
		copyImages(svcSrc, svcDst, imageWorkersNo, dockerWorkersNo)

		if deamon != true {
			break
		}
	}
}
