package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

// DashboardSearchResult represents the structure for dashboard search results
type DashboardSearchResult struct {
	UID         string `json:"uid"`
	Title       string `json:"title"`
	FolderTitle string `json:"folderTitle"`
}

func main() {
	// log 형식 세팅
	log.SetFlags(log.Ltime | log.LstdFlags | log.Llongfile)

	// .env 파일 로드
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Grafana API Key 불러오기
	ALERT_RULES_READ_ONLY_API_KEY := os.Getenv("ALERT_RULES_READ_ONLY_API_KEY")
	grafanaHost := "https://nodeinfra.grafana.net"

	// Get list of all dashboards
	dashboards, err := getAllDashboards(ALERT_RULES_READ_ONLY_API_KEY, grafanaHost)
	if err != nil {
		fmt.Println("Error getting dashboards:", err)
		return
	}

	// Download and save each dashboard JSON model
	for _, dashboard := range dashboards {
		if dashboard.FolderTitle == "LEGACY" || dashboard.FolderTitle == "Templates" {
			saveDashboardJSON(ALERT_RULES_READ_ONLY_API_KEY, grafanaHost, dashboard.UID, dashboard.Title, dashboard.FolderTitle)
		}
	}
}

func getAllDashboards(apiKey, grafanaHost string) ([]DashboardSearchResult, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/search?type=dash-db", grafanaHost), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var dashboards []DashboardSearchResult
	if err := json.Unmarshal(body, &dashboards); err != nil {
		return nil, err
	}

	return dashboards, nil
}

func saveDashboardJSON(apiKey, grafanaHost, uid, title, folderTitle string) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/dashboards/uid/%s", grafanaHost, uid), nil)
	if err != nil {
		fmt.Println("Error creating request for dashboard", title, ":", err)
		return
	}
	req.Header.Add("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error on request execution for dashboard", title, ":", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body for dashboard", title, ":", err)
		return
	}

	folderPath := fmt.Sprintf("./%s/%s", "nodeinfra-grafana-dashboard-json-model", strings.ReplaceAll(folderTitle, " ", "_"))
	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		fmt.Println("Error creating directory for dashboard", folderTitle, ":", err)
		return
	}

	filename := fmt.Sprintf("%s/%s.json", folderPath, strings.ReplaceAll(title, " ", "_"))
	if err := ioutil.WriteFile(filename, body, 0644); err != nil {
		fmt.Println("Error writing JSON to file for dashboard", title, ":", err)
		return
	}

	fmt.Println("Saved dashboard", title, "to", filename)

	commitMessage := "Saved dashboard" + title + "to" + filename
	gitPush(commitMessage)

}

func gitPush(commitMessage string) {
	// 변경할 디렉토리 설정
	repoDir := "nodeinfra-grafana-dashboard-json-model/"
	err := os.Chdir(repoDir)
	if err != nil {
		fmt.Println("Failed to change directory:", err)
		return
	}

	// Git 명령어 실행을 위한 함수
	executeGitCommand := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Git 명령어 순서대로 실행
	if err := executeGitCommand("add", "."); err != nil {
		fmt.Println("Failed to add files:", err)
		return
	}
	if err := executeGitCommand("commit", "-m", commitMessage); err != nil {
		fmt.Println("Failed to commit:", err)
		return
	}
	if err := executeGitCommand("push"); err != nil {
		fmt.Println("Failed to push:", err)
		return
	}

	fmt.Println("Successfully pushed to repository.")
}
