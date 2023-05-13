package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	BaseURL     = "https://api.github.com/repos/"
	ReposList   = "repolist.txt"
	DownloadDir = ".donut-utils"
)

func main() {
	fmt.Println(`     _                   _   
  __| | ___  _ __  _   _| |_ 
 / _' |/ _ \| '_ \| | | | __|
| (_| | (_) | | | | |_| | |_ 
 \__,_|_____|_| |_|\__,_|\__|
 _   _| |_(_| |___           
| | | | __| | / __|          
| |_| | |_| | \__ \          
 \__,_|\__|_|_|___/          
                             `)
	fmt.Println("donut-utils is a collection of cli utilities focusing on convenience and human readable output.\n\nThe applications will be downloaded from Github, and placed in ~/.donut-utils and then ~/.donut-utils will be added to your path.\n\nfor more information, visit the url below:\nhttps://github.com/donuts-are-good/donut-utils\n\nTo abort this process, press CTRL C now.")
	time.Sleep(3 * time.Second)
	data, err := os.ReadFile(ReposList)
	if err != nil {
		fmt.Println("Failed to read repos list file:", err)
		return
	}

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Failed to get current user:", err)
		return
	}

	downloadPath := filepath.Join(usr.HomeDir, DownloadDir)
	err = os.MkdirAll(downloadPath, 0755)
	if err != nil {
		fmt.Println("Failed to create download directory:", err)
		return
	}

	repos := strings.Split(string(data), "\n")

	type appInfo struct {
		Name        string
		Description string
		DownloadURL string
	}
	var availableApps []appInfo

	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}

		// Get repository description
		repoInfoUrl := BaseURL + repo
		resp, err := http.Get(repoInfoUrl)
		if err != nil {
			fmt.Println("Failed to get repository info:", err)
			continue
		}
		if resp.StatusCode != 200 {
			fmt.Println("Received non-200 response code when getting repository info:", resp.StatusCode)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read repository info response body:", err)
			continue
		}
		var repoInfo struct {
			Description string `json:"description"`
		}
		err = json.Unmarshal(body, &repoInfo)
		if err != nil {
			fmt.Println("Failed to unmarshal repository info:", err)
			continue
		}

		repoUrl := BaseURL + repo + "/releases/latest"
		resp, err = http.Get(repoUrl)
		if err != nil {
			fmt.Println("Failed to get latest release:", err)
			continue
		}
		if resp.StatusCode != 200 {
			fmt.Println("Received non-200 response code:", resp.StatusCode)
			continue
		}

		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read response body:", err)
			continue
		}

		var release struct {
			Assets []struct {
				Name               string `json:"name"`
				BrowserDownloadUrl string `json:"browser_download_url"`
			} `json:"assets"`
		}

		err = json.Unmarshal(body, &release)
		if err != nil {
			fmt.Println("Failed to unmarshal release info:", err)
			continue
		}

		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, runtime.GOOS) && strings.Contains(asset.Name, runtime.GOARCH) {
				availableApps = append(availableApps, appInfo{
					Name:        asset.Name,
					Description: repoInfo.Description,
					DownloadURL: asset.BrowserDownloadUrl,
				})
				break
			}
		}
	}

	fmt.Println("\n\n\nThe following applications are available for your system:")
	for i, app := range availableApps {
		fmt.Printf("\n%d. Name: %s\nDescription: %s\n", i+1, app.Name, app.Description)
	}
	fmt.Println("\n\nDo you want to download these applications? (yes/no)")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read user input:", err)
		return
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		for _, app := range availableApps {
			downloadAndStore(app.DownloadURL, downloadPath)
		}
	}
	if runtime.GOOS == "windows" {
		fmt.Println("Please add the following directory to your PATH manually in Windows:")
		fmt.Println(downloadPath)
		fmt.Println("You may need to restart your terminal or system for changes to take effect.")
	} else {
		addToPath(downloadPath)
		fmt.Println("You will need to restart your terminal or source your shell profile for the changes to take effect.")
		fmt.Println("If you're using bash or zsh, you can do this by running one of the following commands:")
		fmt.Println("\nFor bash: source ~/.bashrc")
		fmt.Println("For zsh:  source ~/.zshrc")
	}
}
func downloadAndStore(url string, downloadPath string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Failed to download file:", err)
		return
	}
	defer resp.Body.Close()

	filename := filepath.Base(url)
	index := strings.Index(filename, "-v")
	if index == -1 {
		fmt.Println("Invalid filename format, cannot find version:", filename)
		return
	}

	appName := filename[:index]
	out, err := os.Create(filepath.Join(downloadPath, appName))
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Failed to write file:", err)
		return
	}

	err = os.Chmod(filepath.Join(downloadPath, appName), 0755)
	if err != nil {
		fmt.Println("Failed to change file permissions:", err)
		return
	}

	fmt.Println("File downloaded and saved to:", filepath.Join(downloadPath, appName))
}

func addToPath(dir string) {
	shell := os.Getenv("SHELL")
	var shellrc string

	if strings.Contains(shell, "bash") {
		shellrc = ".bashrc"
	} else if strings.Contains(shell, "zsh") {
		shellrc = ".zshrc"
	} else {
		fmt.Println("Unsupported shell. Please add the following directory to your PATH manually:")
		fmt.Println(dir)
		return
	}

	usr, err := user.Current()
	if err != nil {
		fmt.Println("Failed to get current user:", err)
		return
	}

	shellrcPath := filepath.Join(usr.HomeDir, shellrc)
	file, err := os.OpenFile(shellrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Failed to open shellrc file:", err)
		return
	}

	defer file.Close()

	_, err = file.WriteString("\nexport PATH=$PATH:" + dir)
	if err != nil {
		fmt.Println("Failed to write to shellrc file:", err)
		return
	}

	fmt.Println("Successfully added to PATH in", shellrc)
	fmt.Println("\nTo update your current session, please run the following command:")
	fmt.Printf("\nsource ~/%s\n\n", shellrc)
}
