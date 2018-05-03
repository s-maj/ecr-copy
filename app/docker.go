package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"syscall"
	"time"
)

func getDockerToken(svc *ecr.ECR) (authStr string) {
	var username string
	var password string

	out, err := svc.GetAuthorizationTokenRequest(&ecr.GetAuthorizationTokenInput{}).Send()
	if err != nil {
		log.Error(err.Error())
	}

	for _, data := range out.AuthorizationData {
		decodedToken, _ := base64.StdEncoding.DecodeString(aws.StringValue(data.AuthorizationToken))
		parts := strings.SplitN(string(decodedToken), ":", 2)
		username = parts[0]
		password = parts[1]
	}

	encodedJSON, err := json.Marshal(&types.AuthConfig{Username: username, Password: password})
	if err != nil {
		log.Error(err.Error())
	}

	authStr = base64.URLEncoding.EncodeToString(encodedJSON)
	return
}

func imageCleaner() {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithVersion(dockerApiVer))
	if err != nil {
		log.Error(err.Error())
	}

	for {
		s := syscall.Statfs_t{}
		err := syscall.Statfs("/", &s)
		if err != nil {
			return
		}
		total := int(s.Bsize) * int(s.Blocks)
		used := int(s.Blocks)*int(s.Bsize) - (int(s.Bavail) * int(s.Bsize))

		diskUsage := float64(used) / float64(total)

		if diskUsage >= 0.7 {
			images, err := cli.ImageList(ctx, types.ImageListOptions{})
			if err != nil {
				log.Error(err.Error())
			}

			for _, image := range images {
				_, err := cli.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{Force: true, PruneChildren: true})
				if err != nil {
					log.Error(err.Error())
				}
			}
		}

		time.Sleep(time.Minute)
	}
}

func eventsHandler(events io.ReadCloser, image string) {
	// Create JSON decoder for pull/push events
	d := json.NewDecoder(events)
	var event *DockerEvent

	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}

			log.Error(err.Error())
		} else if event.Error != "" {
			log.Error(fmt.Sprintf("%s: %s", image, event.Error))
		}
	}

	defer events.Close()
}
