package githubapi

import (
        "fmt"
        "io/ioutil"
        "net/http"
        "encoding/json"
        "os"
        "strings"
        structs "alamo-self-diagnostics/structs"
        vault "scc-gitlab-1.dev.octanner.net/octanner/octvault"
)


func GetCommitAuthor(version string) (s string, m string,  e error){

        fmt.Println(version)
        newversion := strings.Replace(version, "https://github.com", "https://api.github.com/repos", -1)
        newerversion := strings.Replace(newversion, "commit", "commits", -1)
        fmt.Println(newerversion)
        req, err := http.NewRequest("GET", newerversion, nil)
        req.Header.Add("Authorization", "token "+vault.GetField(os.Getenv("GITHUB_TOKEN_SECRET"),"token"))
        if err != nil {
                fmt.Println(err)
                return "","", err
        }
        client := http.Client{}
        resp, err := client.Do(req)
        if err != nil {
                fmt.Println(err)
                return "", "",err
        }
        defer resp.Body.Close()
        bodybytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                fmt.Println(err)
                return "", "",err
        }
        fmt.Println(string(bodybytes))
        var commitinfo structs.CommitSpec
        err = json.Unmarshal(bodybytes, &commitinfo)
        if err != nil {
                fmt.Println(err)
                return "", "", err
        }
        fmt.Println(commitinfo.Commit.Author.Name)
//        return commitinfo.Commit.Author.Name, nil
          return commitinfo.Author.Login, commitinfo.Commit.Message, nil


}


