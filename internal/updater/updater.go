package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const defaultAPIURL = "https://api.github.com/repos/cosaria/sing-box/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Update 检查最新版本，如果有新版本则下载并原子替换当前二进制。
func Update(currentVersion string) error {
	fmt.Println("检查最新版本...")
	latest, downloadURL, err := checkLatestVersion(defaultAPIURL)
	if err != nil {
		return fmt.Errorf("检查更新失败: %w", err)
	}
	if !isNewer(currentVersion, latest) {
		fmt.Printf("当前已是最新版本 (%s)\n", currentVersion)
		return nil
	}
	fmt.Printf("发现新版本: %s → %s\n", currentVersion, latest)
	fmt.Println("下载中...")
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取当前二进制路径: %w", err)
	}
	tmpPath := execPath + ".tmp"
	if err := downloadFile(downloadURL, tmpPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("下载失败: %w", err)
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("设置权限失败: %w", err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("替换二进制失败: %w", err)
	}
	fmt.Printf("更新完成: %s → %s\n", currentVersion, latest)
	fmt.Println("请运行 sing-box service restart 完成更新")
	return nil
}

func checkLatestVersion(apiURL string) (version string, downloadURL string, err error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}
	if release.TagName == "" {
		return "", "", fmt.Errorf("GitHub API response missing tag_name")
	}
	archName := fmt.Sprintf("sing-box-linux-%s", runtime.GOARCH)
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, archName) {
			return release.TagName, asset.BrowserDownloadURL, nil
		}
	}
	downloadURL = fmt.Sprintf("https://github.com/cosaria/sing-box/releases/download/%s/%s", release.TagName, archName)
	return release.TagName, downloadURL, nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func isNewer(current, latest string) bool {
	if current == "dev" || current == "" {
		return true
	}
	curParts := parseSemver(current)
	latParts := parseSemver(latest)
	for i := 0; i < 3; i++ {
		if latParts[i] > curParts[i] {
			return true
		}
		if latParts[i] < curParts[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}
