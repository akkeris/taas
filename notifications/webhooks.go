package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"
)

// PromotionResults contains information on a triggered Akkeris app promotion
type PromotionResults struct {
	Message  string `json:"message"`
	Pipeline string `json:"pipeline"`
	From     string `json:"from"`
	To       string `json:"to"`
	Status   string `json:"status"`
}

// WebhookPayload is the structure of the message to be sent as a webhook
type WebhookPayload struct {
	Job              string           `json:"job"`
	Target           string           `json:"target"`
	Status           string           `json:"status"`
	IsPreview        bool             `json:"ispreview"`
	IsCron           bool             `json:"iscron"`
	LogURL           string           `json:"log_url"`
	KibanaURL        string           `json:"kibana_url"`
	ArtifactsURL     string           `json:"artifacts_url"`
	GithubVersion    string           `json:"github_version"`
	RerunURL         string           `json:"rerun_url"`
	CommitAuthor     string           `json:"commit_author"`
	StartTime        string           `json:"start_time"`
	StopTime         string           `json:"stop_time"`
	RunDurationMs    int64            `json:"run_duration_ms"`
	PromotionResults PromotionResults `json:"promotion_results"`
}

// PostWebhooks sends test results to the user-configured webhooks (if any)
func PostWebhooks(diagnostic structs.DiagnosticSpec, status string, promotestatus string, isCron bool, result structs.ResultSpec) error {
	// No configured webhook destinations
	if diagnostic.WebhookURLs == "" {
		return nil
	}

	// Assemble the webhook payload
	payload := WebhookPayload{
		Job:              diagnostic.JobSpace + "/" + diagnostic.Job,
		Target:           diagnostic.App + "-" + diagnostic.Space,
		Status:           status,
		IsPreview:        diagnostic.IsPreview,
		IsCron:           isCron,
		LogURL:           os.Getenv("LOG_URL") + "/logs/" + diagnostic.RunID,
		KibanaURL:        os.Getenv("KIBANA_URL") + "/app/kibana#/doc/logs/logs/run/?id=" + diagnostic.RunID,
		ArtifactsURL:     os.Getenv("ARTIFACTS_URL") + "/v1/artifacts/" + diagnostic.RunID + "/",
		RerunURL:         os.Getenv("RERUN_URL") + "?space=" + diagnostic.Space + "&app=" + diagnostic.App + "&action=" + diagnostic.Action + "&result=" + diagnostic.Result + "&releaseid=" + diagnostic.ReleaseID + "&buildid=" + diagnostic.BuildID,
		CommitAuthor:     diagnostic.CommitAuthor,
		StartTime:        result.Payload.StartTime,
		StopTime:         result.Payload.StopTime,
		RunDurationMs:    result.Payload.BuildTimeMillis,
		PromotionResults: PromotionResults{},
	}

	if diagnostic.GithubVersion != "" {
		payload.GithubVersion = diagnostic.GithubVersion
	}

	if status != "success" {
		payload.PromotionResults.Message = "No promotion triggered - tests not successful"
	} else if diagnostic.PipelineName == "manual" {
		payload.PromotionResults.Message = "No promotion triggered - set to manual"
		payload.PromotionResults.Pipeline = diagnostic.PipelineName
	} else {
		payload.PromotionResults.Message = "Promotion was triggered with result " + promotestatus
		payload.PromotionResults.Status = promotestatus
		payload.PromotionResults.From = diagnostic.TransitionFrom
		payload.PromotionResults.To = diagnostic.TransitionTo
		payload.PromotionResults.Pipeline = diagnostic.PipelineName
	}

	// Send message to each hook URL
	for _, hookURL := range strings.Split(diagnostic.WebhookURLs, ",") {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			fmt.Println(err)
			return err
		}

		if !strings.HasPrefix(hookURL, "http://") && !strings.HasPrefix(hookURL, "https://") {
			hookURL = "https://" + hookURL
		}

		req, err := http.NewRequest("POST", hookURL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println(err)
			return err
		}
		req.Header.Add("Content-type", "application/json")

		client := http.Client{}
		_, err = client.Do(req)
		if err != nil {
			fmt.Println(err)
			return err
		}

		// If we ever want to save the result and do something with it
		// defer resp.Body.Close()
		// bodybytes, err := ioutil.ReadAll(resp.Body)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return err
		// }
		// fmt.Println(string(bodybytes))
		// fmt.Println(resp.status)
	}
	return nil
}
