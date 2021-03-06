package auth

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

func GetUser(req *http.Request) (u string, e error) {
	authheader := strings.Join(req.Header["Authorization"], "")
	req, err := http.NewRequest("GET", os.Getenv("AUTH_HOST")+"/user", nil)
	if err != nil {
		return "nouser", err
	}
	req.Header.Set("Authorization", authheader)
	req.Header.Add("Accept", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "nouser", err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "nouser", err
	}
	var user structs.AuthUser
	err = json.Unmarshal(bodybytes, &user)
	if err != nil {
		return "nouser", err
	}

	utils.PrintDebug("*************************************USER********************************")
	userstr := fmt.Sprintf("%+v\n", user)
	utils.PrintDebug(userstr)
	utils.PrintDebug("*************************************USER********************************")
	if user.Email != "" {
		return user.Email, nil
	}
	if user.Cn != "" {
		return user.Cn, nil
	}
	return "nouser", nil
}
