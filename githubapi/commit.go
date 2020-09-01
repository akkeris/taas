package githubapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"
	"taas/utils"
)

func GetCommitAuthor(version string) (s string, m string, e error) {

	utils.PrintDebug(version)
	newversion := strings.Replace(version, "https://github.com", "https://api.github.com/repos", -1)
	newerversion := strings.Replace(newversion, "commit", "commits", -1)
	utils.PrintDebug(newerversion)
	req, err := http.NewRequest("GET", newerversion, nil)
	req.Header.Add("Authorization", "token "+os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}
	utils.PrintDebug(string(bodybytes))
	var commitinfo structs.CommitSpec
	err = json.Unmarshal(bodybytes, &commitinfo)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}
	utils.PrintDebug(commitinfo.Commit.Author.Name)
	//        return commitinfo.Commit.Author.Name, nil
	return commitinfo.Author.Login, commitinfo.Commit.Message, nil

}
