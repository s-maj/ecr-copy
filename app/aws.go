package app

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func copyRepos(svcSrc, svcDst *ecr.ECR, reposWorkersNo int) {
	controlCh := make(chan struct{})
	repoCh := make(chan string, 100)

	log.Info("Spawning repo workers...")
	for i := 0; i <= reposWorkersNo; i++ {
		go copyReposWorkers(svcDst, repoCh, controlCh)
	}

	log.Info("Looking for repos to copy...")
	req := svcSrc.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{})
	p := req.Paginate()
	for p.Next() {
		page := p.CurrentPage()

		for _, r := range page.Repositories {
			repoCh <- *r.RepositoryName
		}
	}

	for {
		if len(repoCh) == 0 {
			log.Info("Stopping repo workers...")
			close(controlCh)
			return

		} else {
			time.Sleep(time.Second)
		}
	}
}

func copyImages(svcSrc, svcDst *ecr.ECR, imageWorkersNo, dockerWorkersNo int) {
	controlCh := make(chan struct{})
	imageCh := make(chan ecr.ImageDetail, 1000)
	dockerCh := make(chan map[string]string, 10)

	//docker
	srcToken := getDockerToken(svcSrc)
	dstToken := getDockerToken(svcDst)

	log.Info("Spawning docker workers...")
	for i := 0; i <= dockerWorkersNo; i++ {
		go dockerWorkers(srcToken, dstToken, dockerCh, controlCh)
	}

	log.Info("Spawning image workers...")
	for i := 0; i <= imageWorkersNo; i++ {
		go imageWorkers(svcSrc, svcDst, imageCh, dockerCh, controlCh)
	}

	log.Info("Looking for images to copy...")
	req := svcSrc.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{})
	p := req.Paginate()
	for p.Next() {
		page := p.CurrentPage()

		for _, r := range page.Repositories {
			req := svcSrc.DescribeImagesRequest(&ecr.DescribeImagesInput{RepositoryName: r.RepositoryName})
			p := req.Paginate()

			for p.Next() {
				page := p.CurrentPage()

				for _, i := range page.ImageDetails {
					imageCh <- i
				}
			}
		}
	}

	for {
		if (len(imageCh) == 0) && (len(dockerCh) == 0) {
			log.Info("Stopping workers...")
			close(controlCh)
			return

		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

func scheduleImageCopy(svcSrc, svcDst *ecr.ECR, srcImage ecr.ImageDetail, tag string, dockerCh chan<- map[string]string) {
	// Source repository name
	rs, _ := svcSrc.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{*srcImage.RepositoryName},
	}).Send()

	// Destination repository name
	rd, _ := svcDst.DescribeRepositoriesRequest(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{*srcImage.RepositoryName},
	}).Send()

	// Construct a dst/src map for pull/push actions
	dst := fmt.Sprintf("%s:%s", *rd.Repositories[0].RepositoryUri, tag)
	src := fmt.Sprintf("%s:%s", *rs.Repositories[0].RepositoryUri, tag)

	dockerPair := map[string]string{
		"src": src,
		"dst": dst,
	}

	// Put the map to channel for further processing
	dockerCh <- dockerPair

	//Attach original sha256 digest as tag
	dst = fmt.Sprintf("%s:%s", *rd.Repositories[0].RepositoryUri, fmt.Sprintf("orig_sha256_%s", strings.SplitN(*srcImage.ImageDigest, ":", 2)[1]))
	src = fmt.Sprintf("%s:%s", *rs.Repositories[0].RepositoryUri, tag)

	dockerPair = map[string]string{
		"src": src,
		"dst": dst,
	}

	dockerCh <- dockerPair
}

func checkImageSha256(dstImage, srcImage ecr.ImageDetail) bool {
	for _, dstTag := range dstImage.ImageTags {
		if dstTag == fmt.Sprintf("orig_sha256_%s", strings.SplitN(*srcImage.ImageDigest, ":", 2)[1]) {
			return true
		}
	}
	return false
}
