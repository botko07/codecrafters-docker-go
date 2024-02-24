package docker

import (
	"encoding/json"
	"fmt"
	"github.com/codecrafters-io/docker-starter-go/app/util"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	authUrl           = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/%s:pull"
	layerUrl          = "https://registry.hub.docker.com/v2/library/%s/blobs/%s"
	manifestUrl       = "https://registry.hub.docker.com/v2/library/%s/manifests/%s"
	manifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
)

type AuthRegistry struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}
type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
}

var (
	client     *http.Client
	clientOnce sync.Once
)

func getHTTPClient() *http.Client {
	clientOnce.Do(func() {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	})
	return client
}
func getAuthToken(image string) *AuthRegistry {
	client := getHTTPClient()
	res, err := client.Get(fmt.Sprintf(authUrl, image))
	util.ExitIfErr(err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	util.ExitIfErr(err)
	var auth AuthRegistry
	err = json.Unmarshal(body, &auth)
	util.ExitIfErr(err)
	return &auth
}
func getManifest(image string, version string, auth *AuthRegistry) *Manifest {
	client := getHTTPClient()
	req, err := http.NewRequest("GET", fmt.Sprintf(manifestUrl, image, version), nil)
	util.ExitIfErr(err)
	req.Header.Add("Accept", manifestMediaType)
	req.Header.Add("Authorization", "Bearer "+auth.AccessToken)
	res, err := client.Do(req)
	util.ExitIfErr(err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("Manifest request failed, request status code: %d", res.StatusCode)
		return nil
	}
	body, err := io.ReadAll(res.Body)
	util.ExitIfErr(err)
	var manifest Manifest
	err = json.Unmarshal(body, &manifest)
	util.ExitIfErr(err)
	return &manifest
}
func pullLayer(auth *AuthRegistry, url string, filename string) {
	file, err := os.Create(filename)
	util.ExitIfErr(err)
	defer file.Close()
	client := getHTTPClient()
	req, err := http.NewRequest("GET", url, nil)
	util.ExitIfErr(err)
	req.Header.Add("Accept", manifestMediaType)
	req.Header.Add("Authorization", "Bearer "+auth.AccessToken)
	resp, err := client.Do(req)
	util.ExitIfErr(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Pulling layer from %s return status code %d", url, resp.StatusCode)
	}
	_, err = io.Copy(file, resp.Body)
	util.ExitIfErr(err)
}
func extract(fileName string, destination string) {
	cmd := exec.Command("tar", "xzf", fileName, "-C", destination)
	err := cmd.Run()
	util.ExitIfErr(err)
	err = os.Remove(fileName)
	util.ExitIfErr(err)
}
func Pull(image string, destination string) {
	img, ver, ok := strings.Cut(image, ":")
	if !ok {
		ver = "latest"
	}
	auth := getAuthToken(img)
	manifest := getManifest(img, ver, auth)
	for idx, layer := range manifest.Layers {
		sourceUrl := fmt.Sprintf(layerUrl, img, layer.Digest)
		fileName := filepath.Join(destination, fmt.Sprintf("layer-%d.tar", idx))
		pullLayer(auth, sourceUrl, fileName)
		extract(fileName, destination)
	}
}
