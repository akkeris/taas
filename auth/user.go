package auth

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"
)

func GetUser(req *http.Request) (u string, e error) {
	authheader := strings.Join(req.Header["Authorization"], "")
	req, err := http.NewRequest("GET", os.Getenv("AUTH_HOST")+"/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authheader)
        req.Header.Add("Accept", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var user structs.AuthUser
	err = json.Unmarshal(bodybytes, &user)
	if err != nil {
		return "", err
	}
	return user.Email, nil
}
