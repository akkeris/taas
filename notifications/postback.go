package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"
)

func PostResults(result structs.ResultSpec) (e error) {
	if os.Getenv("POSTBACKURL") != "none" && os.Getenv("POSTBACKURL") != "" {
		p, err := json.Marshal(result)
		if err != nil {
			fmt.Println(err)
			return err
		}
		postbackurl := os.Getenv("POSTBACKURL")
		postbacks := strings.Split(postbackurl, ",")
		for _, postback := range postbacks {
			fmt.Println("Posting to " + postback)
			req, err := http.NewRequest("POST", postback, bytes.NewBuffer(p))
			if err != nil {
				fmt.Println(err)
				return err
			}
			req.Header.Add("Content-type", "application/json")
			client := http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
				return err
			}
			defer resp.Body.Close()
			bodybytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println("Response body from postback: " + string(bodybytes))
			fmt.Println("Response code from postback: " + resp.Status)
		}
	}
	return nil
}
