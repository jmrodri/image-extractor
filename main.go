package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	registryimage "github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	log "github.com/sirupsen/logrus"
)

func DiscardLogger() *log.Entry {
	logger := log.New()
	logger.SetOutput(ioutil.Discard)
	return log.NewEntry(logger)
}

func StdoutLogger() *log.Entry {
	logger := log.New()
	logger.SetOutput(os.Stdout)
	return log.NewEntry(logger)
}

func ExtractImage(ctx context.Context, logger *log.Entry, image string) (string, error) {
	if logger == nil {
		logger = DiscardLogger()
	}

	// Use a temp directory for image files.
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	imageDir, err := ioutil.TempDir(wd, "image-")
	if err != nil {
		return "", err
	}
	// This should always work, but if it doesn't imageDir is still valid.
	if dir, err := filepath.Rel(wd, imageDir); err == nil {
		imageDir = dir
	}
	// Export the image into imageDir
	logger = logger.WithFields(log.Fields{"dir": imageDir})

	// Use a containerd registry instead of shelling out to a container tooll.
	reg, err := containerdregistry.NewRegistry(containerdregistry.WithLog(logger))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := reg.Destroy(); err != nil {
			logger.WithError(err).Warn("Error destroying local cache")
		}
	}()

	// Pull the image if it isn't present locally
	// if !local {
	if err := reg.Pull(ctx, registryimage.SimpleReference(image)); err != nil {
		return "", fmt.Errorf("error pulling image %s: %v", image, err)
	}
	// }

	// Unpack the image's contents.
	if err := reg.Unpack(ctx, registryimage.SimpleReference(image), imageDir); err != nil {
		return "", fmt.Errorf("error unpacking image %s: %v", image, err)
	}

	return imageDir, nil
}

func main() {

	if len(os.Args[1:]) < 1 {
		fmt.Println("image-extractor <imagespec>")
		os.Exit(-1)
	}

	image := os.Args[1]
	outputDir, err := ExtractImage(context.Background(), StdoutLogger(), image)
	if err != nil {
		fmt.Printf("Error loading image: %v\n", err)
		os.Exit(-1)
	}
	fmt.Printf("Image extracted to %v\n", outputDir)
}
