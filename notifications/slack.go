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
	"time"
)

type Attachment struct {
	Fallback  string `json:"fallback"`
	Pretext   string `json:"pretext"`
	Title     string `json:"title"`
	TitleLink string `json:"title_link"`
	Text      string `json:"text"`
	Color     string `json:"color"`
	Ts        int32  `json:"ts"`
}

type Slack struct {
	Channel     string       `json:"channel"`
	Username    string       `json:"username"`
	Text        string       `json:"text"`
	UnfurlLinks bool         `json:"unfurl_links"`
	UnfurlMedia bool         `json:"unfurl_media"`
	IconEmoji   string       `json:"icon_emoji"`
	Attachments []Attachment `json:"attachments"`
}

func PostToSlack(diagnostic structs.DiagnosticSpec, status string) {
	var slack Slack
	testframework := strings.ToUpper(strings.Split(strings.Replace(diagnostic.Image, "quay.octanner.io/developer/", "", -1), ":")[0])

	fmt.Println("********************************")
	fmt.Println(testframework)
	fmt.Println("********************************")
	if diagnostic.Slackchannel != "" {
		slack.Channel = diagnostic.Slackchannel
	}
	if diagnostic.Slackchannel == "" {
		slack.Channel = "#" + os.Getenv("SLACK_NOTIFICATION_CHANNEL_DEFAULT")
	}

	slackurl := os.Getenv("SLACK_NOTIFICATION_URL")

	slack.Username = "Test Results"

	slack.Text = "Job: " + diagnostic.JobSpace + "/" + diagnostic.Job
	slack.Text = slack.Text + "  Testing: " + diagnostic.App + "-" + diagnostic.Space
	slack.Text = slack.Text + "  Status: " + status + "  \n"
	slack.Text = slack.Text + "<" + os.Getenv("LOG_URL") + "/logs/" + diagnostic.RunID + "|Logs>   "
	slack.Text = slack.Text + "<" + os.Getenv("KIBANA_URL") + "/app/kibana#/doc/logs/logs/run/?id=" + diagnostic.RunID + "|Kibana>  "
        slack.Text = slack.Text + "<" + os.Getenv("ARTIFACTS_URL") + "/v1/artifacts/"+diagnostic.RunID+"/ |Artifacts>  "
	if diagnostic.GithubVersion != "" {
		slack.Text = slack.Text + "<" + diagnostic.GithubVersion + "|Commit>  "
	}
	slack.Text = slack.Text + "<" + os.Getenv("RERUN_URL") + "?space=" + diagnostic.Space + "&app=" + diagnostic.App + "&action=" + diagnostic.Action + "&result=" + diagnostic.Result + "&buildid=" + diagnostic.BuildID + "|Rerun>\n"
	slack.Text = slack.Text + "Changes Made by: @" + diagnostic.CommitAuthor
	slack.UnfurlLinks = false
	slack.UnfurlMedia = false
	var attachments []Attachment
	var attachment Attachment

	if status == "success" {
		slack.IconEmoji = ":metal:"
		attachment.Color = "good"
		attachment.Text = "OK"
	} else {
		slack.IconEmoji = ":poop:"
		attachment.Color = "danger"
		attachment.Text = "FAIL"
	}

	attachment.Ts = int32(time.Now().Unix())

	attachments = append(attachments, attachment)
	slack.Attachments = attachments

	p, err := json.Marshal(slack)
	if err != nil {
		fmt.Println(err)

	}
	fmt.Println(slack.Text)

	if os.Getenv("ENABLE_SLACK_NOTIFICATIONS") == "true" {
		req, err := http.NewRequest("POST", slackurl, bytes.NewBuffer(p))
		if err != nil {
			fmt.Println(err)
		}
		req.Header.Add("Content-type", "application/json")
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
		}
		defer resp.Body.Close()
		bodybytes, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(bodybytes))
		PostPromoteToSlack(diagnostic, status)
	}
}

func PostPromoteToSlack(diagnostic structs.DiagnosticSpec, status string) {
	var slack Slack
	//testframework:=strings.ToUpper(strings.Split(strings.Replace(diagnostic.Image,"quay.octanner.io/developer/","",-1),":")[0])
	if diagnostic.Slackchannel != "" {
		slack.Channel = diagnostic.Slackchannel
	}
	if diagnostic.Slackchannel == "" {
		slack.Channel = "#" + os.Getenv("SLACK_NOTIFICATION_CHANNEL_DEFAULT")
	}

	slackurl := os.Getenv("SLACK_NOTIFICATION_URL")
	slack.Username = "Promotion Action"

	if status == "success" && diagnostic.PipelineName != "manual" {
		slack.Text = "Promotion from " + diagnostic.TransitionFrom + " to " + diagnostic.TransitionTo + " triggered"
		slack.IconEmoji = ":mortar_board:"
	}
	if status == "success" && diagnostic.PipelineName == "manual" {
		slack.Text = "No promotion triggered - set to manual"
		slack.IconEmoji = ":keyboard:"
	}
	if status != "success" {
		slack.Text = "No promotion triggered - tests not successful"
		slack.IconEmoji = ":kaboom:"
	}

	p, err := json.Marshal(slack)
	if err != nil {
		fmt.Println(err)

	}
	fmt.Println(slack.Text)

	if os.Getenv("ENABLE_SLACK_NOTIFICATIONS") == "true" {
		req, err := http.NewRequest("POST", slackurl, bytes.NewBuffer(p))
		if err != nil {
			fmt.Println(err)
		}
		req.Header.Add("Content-type", "application/json")
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
		}
		defer resp.Body.Close()
		bodybytes, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(bodybytes))
	}

}
