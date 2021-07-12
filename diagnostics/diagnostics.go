package diagnostics

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strconv"
	"strings"
	artifacts "taas/artifacts"
	dbstore "taas/dbstore"
	diagnosticlogs "taas/diagnosticlogs"
	githubapi "taas/githubapi"
	akkeris "taas/jobs"
	jobs "taas/jobs"
	notifications "taas/notifications"
	pipelines "taas/pipelines"
	structs "taas/structs"
	"taas/utils"
	"text/template"
	"time"

	"github.com/go-martini/martini"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
)

func RunDiagnostic(diagnostic structs.DiagnosticSpec, isCron bool, cronjob structs.Cronjob) (e error) {

	// may need to inject the run id into the config set at this point so that it is available to internal code if it will send logs

	if isCron {
		runiduuid, _ := uuid.NewV4()
		runid := runiduuid.String()
		diagnostic.RunID = runid
		diagnostic.Startdelay = 1
		diagnostic.Command = cronjob.Command
	}

	var newvar structs.Varspec
	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "DIAGNOSTIC_RUNID"
	newvar.Varvalue = diagnostic.RunID
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "TAAS_RUNID"
	newvar.Varvalue = diagnostic.RunID
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "TAAS_ARTIFACT_REGION"
	newvar.Varvalue = os.Getenv("AWS_REGION")
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "TAAS_AWS_ACCESS_KEY_ID"
	newvar.Varvalue = os.Getenv("AWS_ACCESS_KEY_ID")
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "TAAS_AWS_SECRET_ACCESS_KEY"
	newvar.Varvalue = os.Getenv("AWS_SECRET_ACCESS_KEY")
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "TAAS_ARTIFACT_BUCKET"
	newvar.Varvalue = os.Getenv("AWS_S3_BUCKET")
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	// These environment variables tell the test suite that it is running in Akkeris CI
	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "AKKERIS_CI"
	newvar.Varvalue = "true"
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "CI"
	newvar.Varvalue = "true"
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	newvar.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	newvar.Varname = "ORIGIN"
	newvar.Varvalue = "taas"
	akkeris.AddVar(newvar)
	akkeris.UpdateVar(newvar)

	go check(diagnostic, isCron, cronjob)
	return nil
}

func getStatusCheck(diagnostic structs.DiagnosticSpec) (c string, e error) {
	req, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses", nil)
	req.Header.Add("Authorization", "Bearer "+diagnostic.Token)
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	var bodybytes []byte
	if resp.StatusCode == 401 {
		req2, err := http.NewRequest("GET", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses", nil)
		req2.Header.Add("Content-type", "application/json")
		req2.Header.Set("Authorization", os.Getenv("APP_CONTROLLER_AUTH"))
		client2 := http.Client{}
		resp2, err := client2.Do(req2)
		if err != nil {
			fmt.Println(err)
			return "", err
		}
		defer resp2.Body.Close()
		bodybytes, err = ioutil.ReadAll(resp2.Body)
		utils.PrintDebug(string(bodybytes))
	} else {
		defer resp.Body.Close()
		bodybytes, err = ioutil.ReadAll(resp.Body)
		utils.PrintDebug(string(bodybytes))
	}
	var statuses structs.Statuses
	var statusid string
	err = json.Unmarshal(bodybytes, &statuses)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	for _, status := range statuses.Statuses {
		utils.PrintDebug(status.ID)
		testContext := "taas/" + diagnostic.Job + "-" + diagnostic.JobSpace
		if len(testContext) >= 32 {
			testContext = testContext[:28] + "..."
		}
		if status.Context == testContext {
			statusid = status.ID
		}
	}
	return statusid, nil
}

func updateStatusCheck(statusid string, releasestatus structs.ReleaseStatus, diagnostic structs.DiagnosticSpec, loglink string) (e error) {
	p, err := json.Marshal(releasestatus)
	if err != nil {
		fmt.Println(err)
		return err
	}
	utils.PrintDebug("SETTING STATUS CHECK: " + releasestatus.State)
	req, err := http.NewRequest("PATCH", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses/"+statusid, bytes.NewBuffer(p))
	req.Header.Add("Authorization", "Bearer "+diagnostic.Token)
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var bodybytes []byte
	if resp.StatusCode == 401 {
		req2, err := http.NewRequest("PATCH", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses/"+statusid, bytes.NewBuffer(p))
		req2.Header.Add("Content-type", "application/json")
		req2.Header.Set("Authorization", os.Getenv("APP_CONTROLLER_AUTH"))
		client2 := http.Client{}
		resp2, err := client2.Do(req2)
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer resp2.Body.Close()
		bodybytes, err = ioutil.ReadAll(resp2.Body)
		utils.PrintDebug(string(bodybytes))
		return nil
	} else {
		defer resp.Body.Close()
		bodybytes, err = ioutil.ReadAll(resp.Body)
		utils.PrintDebug(string(bodybytes))
		return nil
	}
}

func createStatusCheck(releasestatus structs.ReleaseStatus, diagnostic structs.DiagnosticSpec, loglink string) (e error) {
	p, err := json.Marshal(releasestatus)
	if err != nil {
		fmt.Println(err)
		return err
	}
	utils.PrintDebug("SETTING STATUS CHECK: " + releasestatus.State)
	req, err := http.NewRequest("POST", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses", bytes.NewBuffer(p))
	req.Header.Add("Authorization", "Bearer "+diagnostic.Token)
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var bodybytes []byte
	if resp.StatusCode == 401 {
		req2, err := http.NewRequest("POST", os.Getenv("APP_CONTROLLER_URL")+"/apps/"+diagnostic.App+"-"+diagnostic.Space+"/releases/"+diagnostic.ReleaseID+"/statuses", bytes.NewBuffer(p))
		req.Header.Add("Content-type", "application/json")
		req2.Header.Set("Authorization", os.Getenv("APP_CONTROLLER_AUTH"))
		client2 := http.Client{}
		resp2, err := client2.Do(req2)
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer resp2.Body.Close()
		bodybytes, err := ioutil.ReadAll(resp2.Body)
		utils.PrintDebug(string(bodybytes))
		return nil
	} else {
		defer resp.Body.Close()
		bodybytes, err = ioutil.ReadAll(resp.Body)
		utils.PrintDebug(string(bodybytes))
		return nil
	}
}

func setStatusCheck(status string, diagnostic structs.DiagnosticSpec, loglink string) {
	var releasestatus structs.ReleaseStatus
	releasestatus.Name = "TaaS Tests"
	if status == "success" {
		releasestatus.State = "success"
		releasestatus.Description = "Tests Passed!"
		releasestatus.TargetURL = loglink
		releasestatus.ImageURL = os.Getenv("ARTIFACTS_URL") + "/success.png"
		statusid, err := getStatusCheck(diagnostic)
		if err != nil {
			fmt.Println(err)
		}
		utils.PrintDebug("Updating status: " + statusid)
		updateStatusCheck(statusid, releasestatus, diagnostic, loglink)
	}
	if status == "pending" {
		releasestatus.State = "pending"
		releasestatus.Description = "Tests are still running"
		releasestatus.Context = "taas/" + diagnostic.Job + "-" + diagnostic.JobSpace
		if len(releasestatus.Context) >= 32 {
			releasestatus.Context = releasestatus.Context[:28] + "..."
		}
		createStatusCheck(releasestatus, diagnostic, loglink)
		statusid, err := getStatusCheck(diagnostic)
		if err != nil {
			fmt.Println(err)
		}
		utils.PrintDebug("Updating status: " + statusid)
		releasestatus.Context = ""
		updateStatusCheck(statusid, releasestatus, diagnostic, loglink)
	}
	if status == "timedout" {
		releasestatus.State = "failure"
		releasestatus.Description = "Tests Timed Out"
		releasestatus.TargetURL = loglink
		releasestatus.ImageURL = os.Getenv("ARTIFACTS_URL") + "/fail.png"
		statusid, err := getStatusCheck(diagnostic)
		if err != nil {
			fmt.Println(err)
		}
		updateStatusCheck(statusid, releasestatus, diagnostic, loglink)
	}
	if status == "failed" {
		releasestatus.State = "failure"
		releasestatus.Description = "Tests Failed"
		releasestatus.TargetURL = loglink
		releasestatus.ImageURL = os.Getenv("ARTIFACTS_URL") + "/fail.png"
		statusid, err := getStatusCheck(diagnostic)
		if err != nil {
			fmt.Println(err)
		}
		updateStatusCheck(statusid, releasestatus, diagnostic, loglink)
	}
}

func check(diagnostic structs.DiagnosticSpec, isCron bool, cronjob structs.Cronjob) {
	// Create status check (if applicable)
	if os.Getenv("STATUS_CHECKS") == "true" && !isCron {
		setStatusCheck("pending", diagnostic, "")
	}

	if isCron {
		fmt.Println("\n[" + diagnostic.RunID + "]: Starting cron job for " + diagnostic.Job + "-" + diagnostic.JobSpace + " (cronid: " + cronjob.ID + ")")
	} else {
		fmt.Println("\n[" + diagnostic.RunID + "]: Starting job for " + diagnostic.Job + "-" + diagnostic.JobSpace)
	}

	// Delay start of job according to the configured start delay
	utils.PrintDebug("Job " + diagnostic.RunID + " start delay set to : " + strconv.Itoa(diagnostic.Startdelay))
	time.Sleep(time.Second * time.Duration(diagnostic.Startdelay))

	// Configure the Kubernetes pod
	var oneoff structs.OneOffSpec
	oneoff.Space = diagnostic.JobSpace
	oneoff.Podname = strings.ToLower(diagnostic.Job) + "-" + diagnostic.RunID
	oneoff.Containername = strings.ToLower(diagnostic.Job)
	if strings.HasPrefix(diagnostic.Image, "akkeris://") {
		imageappname := strings.Replace(diagnostic.Image, "akkeris://", "", -1)
		currentimage := akkeris.GetCurrentImage(imageappname)
		oneoff.Image = currentimage
		diagnostic.Image = currentimage
	} else {
		utils.PrintDebug("assuming docker image url")
		oneoff.Image = diagnostic.Image
	}
	oneoff.Command = diagnostic.Command
	fetched, err := akkeris.GetVars(diagnostic.Job, diagnostic.JobSpace)
	if err != nil {
		fmt.Println(err)
	}
	oneoff.Env = fetched

	// Allow users to set `PREVIEW_URL_VAR` to the name of the config var that they want
	// us to inject the URL of the preview app into

	var injectvarname string

	if diagnostic.IsPreview {
		// Find the PREVIEW_URL_VAR to replace
		for _, element := range oneoff.Env {
			if element.Name == "PREVIEW_URL_VAR" {
				injectvarname = element.Value
				break
			}
		}
		// Replace the target config var with the URL of the preview app
		for i, element := range oneoff.Env {
			if element.Name == injectvarname {
				oneoff.Env[i].Value = "http://" + diagnostic.App + "." + diagnostic.Space + ".svc.cluster.local"
			}
		}
	}

	// Create the job in the database with status of "starting"
	diagnostic.OverallStatus = "starting"
	if isCron {
		dbstore.StoreCronRun(diagnostic, time.Now().UTC(), nil, cronjob.ID)
	} else {
		dbstore.StoreRun(diagnostic)
	}

	// Delete any identical old pods that may exist
	akkeris.Deletepod(oneoff.Space, oneoff.Podname)
	time.Sleep(time.Second * 5)

	// Create the pod in Kubernetes, starting the run
	_, err = akkeris.Startpod(oneoff)
	starttime := time.Now().UTC()
	endtime := time.Now().UTC()

	// Set this to "timeout" to consider the job timed out if the loop ends without changing status
	overallstatus := "timedout"

	var instance string
	var loglines structs.LogLines
	var i float64

	if err != nil {
		utils.PrintDebug("JOB " + diagnostic.RunID + " FAILED: unable to start pod")
		overallstatus = "failed"
		endtime = time.Now().UTC()
		loglines.Logs = append(loglines.Logs, "Message: Unable to start tests")
		loglines.Logs = append(loglines.Logs, "")
		loglines.Logs = append(loglines.Logs, "Message: "+err.Error())
	} else {
		time.Sleep(time.Second * 3)
		updated := false

		// Check on the Kubernetes pod until the configured timeout
		for i = 0.0; i < float64(diagnostic.Timeout); i += 0.333 {
			time.Sleep(time.Millisecond * 333)
			akkerisapiurl := os.Getenv("AKKERIS_API_URL")
			req, err := http.NewRequest("GET", akkerisapiurl+"/v1/space/"+diagnostic.JobSpace+"/app/"+oneoff.Podname+"/instance", nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			req.Header.Add("Content-type", "application/json")
			client := http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()
			bodybytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}

			var status structs.InstanceStatusSpec
			err = json.Unmarshal(bodybytes, &status)
			if err != nil {
				fmt.Println(err)
				return
			}
			if len(status) > 1 {
				utils.PrintDebug("JOB " + diagnostic.RunID + " FAILED")
				overallstatus = "failed"
				endtime = time.Now().UTC()
				for _, element := range status {
					if element.Reason == "Error" || element.Phase == "Running/terminated" || element.Phase == "Failed/terminated" {
						utils.PrintDebug(element.Instanceid)
						instance = element.Instanceid
						utils.PrintDebug(element.Phase)
						utils.PrintDebug(element.Reason)
					}
				}
				break
			}
			if len(status) == 0 {
				continue
			}
			instance = status[0].Instanceid
			if status[0].Phase == "Succeeded/terminated" && status[0].Reason == "Completed" {
				utils.PrintDebug("JOB FINISHED OK")
				overallstatus = "success"
				endtime = time.Now().UTC()
				break
			}

			if status[0].Phase == "Running/terminated" && status[0].Reason == "Error" {
				utils.PrintDebug("JOB FAILED")
				overallstatus = "failed"
				endtime = time.Now().UTC()
				break
			}

			if status[0].Phase == "Running/waiting" && status[0].Reason == "CrashLoopBackOff" {
				utils.PrintDebug("JOB FAILED")
				overallstatus = "failed"
				endtime = time.Now().UTC()
				break
			}
			if status[0].Phase == "Failed/terminated" && status[0].Reason == "Error" {
				utils.PrintDebug("JOB FAILED")
				overallstatus = "failed"
				endtime = time.Now().UTC()
				break
			}
			if status[0].Phase == "Failed/terminated" && status[0].Reason == "ContainerCannotRun" {
				utils.PrintDebug("JOB FAILED")
				overallstatus = "failed"
				endtime = time.Now().UTC()
				break
			}

			// Update the job in the database to show that the pod is running
			if status[0].Phase == "Running/running" && status[0].Reason == "" && !updated {
				diagnostic.OverallStatus = "running"
				if isCron {
					dbstore.UpdateCronRun(diagnostic, nil)
				} else {
					dbstore.UpdateRunStatus(diagnostic)
				}
				updated = true
			}
		}
	}

	fmt.Println("[" + diagnostic.RunID + "]: Finishing job for " + diagnostic.Job + "-" + diagnostic.JobSpace)

	// Store logs
	logs, err := jobs.GetTestLogs(diagnostic.JobSpace, diagnostic.Job, instance)
	if err != nil {
		fmt.Println(err)
	}
	diagnostic.OverallStatus = overallstatus
	loglines.Logs = append(loglines.Logs, logs...)
	diagnosticlogs.WriteLogES(diagnostic, loglines)
	_, err = describePodAndUploadToS3(diagnostic.JobSpace, oneoff.Podname, diagnostic.RunID)
	if err != nil {
		fmt.Println(err)
	}

	// Update job in the database with the final status
	if isCron {
		err = dbstore.UpdateCronRun(diagnostic, &endtime)
	} else {
		err = dbstore.UpdateRunStatus(diagnostic)
	}
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("[" + diagnostic.RunID + "]: Job finished with status \"" + overallstatus + "\"")

	// Notify postback URL
	var result structs.ResultSpec
	result.Payload.Lifecycle = "finished"
	result.Payload.Outcome = overallstatus
	result.Payload.Status = overallstatus
	result.Payload.StartTime = starttime.Format(time.RFC3339)
	result.Payload.StopTime = endtime.Format(time.RFC3339)
	var duration = endtime.Sub(starttime)
	result.Payload.BuildTimeMillis = duration.Nanoseconds() / 1e6

	var step structs.StepSpec
	step.Name = diagnostic.Job + "-" + diagnostic.JobSpace
	step.Organization = diagnostic.Organization
	var action structs.ActionSpec
	action.Name = diagnostic.Job + "-" + diagnostic.JobSpace
	action.Status = overallstatus
	var actions []structs.ActionSpec
	actions = append(actions, action)
	step.Actions = actions

	var steps []structs.StepSpec
	steps = append(steps, step)
	result.Payload.Steps = steps
	if err != nil {
		fmt.Println(err)
	}
	notifications.PostResults(result)

	// Update status check (if applicable)
	if os.Getenv("STATUS_CHECKS") == "true" {
		loglink := os.Getenv("LOG_URL") + "/logs/" + diagnostic.RunID
		if !isCron {
			setStatusCheck(overallstatus, diagnostic, loglink)
		}
	}

	// Handle pipeline promotion (if applicable)
	var promotestatus string
	promotestatus = "failed"
	if overallstatus == "success" && diagnostic.PipelineName != "manual" {
		if isCron != true {
			transitionfrom := diagnostic.TransitionFrom
			transitionto := diagnostic.TransitionTo
			transitiontoa := strings.Split(transitionto, ",")
			utils.PrintDebug("Promoting " + transitionfrom + " to " + transitionto + " for job " + diagnostic.RunID)

			var fromappid string
			var toappids []string
			var pipelineid string
			pipeline, err := pipelines.GetPipeline(diagnostic.PipelineName)
			if err != nil {
				fmt.Println(err)
			}
			for _, element := range pipeline {
				if element.Stage+":"+element.App.Name == transitionfrom {
					fromappid = element.App.ID
					pipelineid = element.Pipeline.ID
				}
				for _, trelement := range transitiontoa {
					if element.Stage+":"+element.App.Name == trelement {
						toappids = append(toappids, element.App.ID)
					}
				}
			}
			var promotion structs.PromotionSpec
			var targets []structs.Target

			for _, appid := range toappids {
				var target structs.Target
				target.App.ID = appid
				targets = append(targets, target)
			}
			promotion.Targets = targets
			promotion.Pipeline.ID = pipelineid
			promotion.Source.App.ID = fromappid
			promotestatus, err = pipelines.PromoteApp(promotion)
			if err != nil {
				fmt.Println(err)
			}
			utils.PrintDebug("Promotion finished with status \"" + promotestatus + "\" for job " + diagnostic.RunID)
		}
	}

	// Post results to Slack & any webhooks, and clean up Kubernetes pod
	notifications.PostToSlack(diagnostic, overallstatus, promotestatus, isCron)
	notifications.PostWebhooks(diagnostic, overallstatus, promotestatus, isCron, result)
	akkeris.Deletepod(oneoff.Space, oneoff.Podname)
	return
}

func scaleToZero(diagnostic structs.DiagnosticSpec) (e error) {

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1beta1/space/"+diagnostic.JobSpace+"/jobs/"+diagnostic.Job+"/scale/0/1", nil)
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
	utils.PrintDebug(string(bodybytes))

	return nil

}
func GetDiagnostics(space string, app string, action string, result string) (d []structs.DiagnosticSpec, e error) {
	var diagnostics []structs.DiagnosticSpec
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return diagnostics, dberr
	}
	defer db.Close()
	stmt, err := db.Prepare("select id, space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay,slackchannel,coalesce(command,null,''), coalesce(testpreviews,null,false), coalesce(ispreview,null,false), coalesce(webhookurls,null,'') from diagnostics where space = $1 and app = $2 and action = $3 and result=$4")
	if err != nil {
		fmt.Println(err)
		return diagnostics, err
	}
	var did string
	var dspace string
	var dapp string
	var daction string
	var dresult string
	var djob string
	var djobspace string
	var dimage string
	var dpipelinename string
	var dtransitionfrom string
	var dtransitionto string
	var dtimeout int
	var dstartdelay int
	var dslackchannel string
	var dcommand string
	var dtestpreviews bool
	var dispreview bool
	var dwebhookurls string

	defer stmt.Close()
	rows, err := stmt.Query(space, app, action, result)
	for rows.Next() {
		err := rows.Scan(&did, &dspace, &dapp, &daction, &dresult, &djob, &djobspace, &dimage, &dpipelinename, &dtransitionfrom, &dtransitionto, &dtimeout, &dstartdelay, &dslackchannel, &dcommand, &dtestpreviews, &dispreview, &dwebhookurls)
		if err != nil {
			fmt.Println(err)
			return diagnostics, err
		}
		var diagnostic structs.DiagnosticSpec
		diagnostic.ID = did
		diagnostic.Space = dspace
		diagnostic.App = dapp
		diagnostic.Action = daction
		diagnostic.Result = dresult
		diagnostic.Job = djob
		diagnostic.JobSpace = djobspace
		diagnostic.Image = dimage
		diagnostic.PipelineName = dpipelinename
		diagnostic.TransitionFrom = dtransitionfrom
		diagnostic.TransitionTo = dtransitionto
		diagnostic.Timeout = dtimeout
		diagnostic.Startdelay = dstartdelay
		diagnostic.Slackchannel = dslackchannel
		diagnostic.Command = dcommand
		diagnostic.TestPreviews = dtestpreviews
		diagnostic.IsPreview = dispreview
		diagnostic.WebhookURLs = dwebhookurls
		runiduuid, _ := uuid.NewV4()
		runid := runiduuid.String()
		utils.PrintDebug(runid)
		diagnostic.RunID = runid
		diagnostics = append(diagnostics, diagnostic)
	}

	db.Close()

	return diagnostics, nil

}

func HTTPDeleteDiagnostic(req *http.Request, params martini.Params, r render.Render) {
	diagnostic, err := dbstore.FindDiagnostic(params["provided"])
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}

	err = DeleteDiagnostic(diagnostic)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return

	}
	dbstore.AddDiagnosticDeleteAudit(req, diagnostic)
	r.JSON(200, map[string]interface{}{"status": "deleted"})

}

func DeleteDiagnostic(diagnostic structs.DiagnosticSpec) (e error) {

	err := akkeris.DeleteService(diagnostic)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = akkeris.DeleteBind(diagnostic)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = akkeris.DeleteSet(diagnostic)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = akkeris.DeleteJob(diagnostic)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func CreateDiagnostic(req *http.Request, diagnosticspec structs.DiagnosticSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
		return
	}

	d, err := dbstore.FindDiagnostic(diagnosticspec.Job + "-" + diagnosticspec.JobSpace)
	if err == nil && d.ID != "" {
		r.Text(400, "A diagnostic with the given name and space already exists.")
		return
	} else if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}

	isvalidspace, err := akkeris.IsValidSpace(diagnosticspec.JobSpace)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if !isvalidspace {
		r.JSON(500, map[string]interface{}{"response": "Invalid Space"})
		return
	}
	if diagnosticspec.Startdelay == 0 {
		defaultstartdelay, _ := strconv.Atoi(os.Getenv("DEFAULT_START_DELAY"))
		diagnosticspec.Startdelay = defaultstartdelay
	}
	newappiduuid, _ := uuid.NewV4()
	newappid := newappiduuid.String()
	diagnosticspec.ID = newappid
	err = createDiagnostic(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return

	}
	dbstore.AddDiagnosticCreateAudit(req, diagnosticspec)
	r.JSON(200, map[string]interface{}{"status": "created"})
}

func createDiagnostic(diagnosticspec structs.DiagnosticSpec) (e error) {
	err := akkeris.CreateConfigSet(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = akkeris.CreateVariables(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = akkeris.CreateBind(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = akkeris.CreateService(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = akkeris.CreateHooks(diagnosticspec.App + "-" + diagnosticspec.Space)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Add hooks to run diagnostic on preview apps if the "TestPreviews" property is true
	if diagnosticspec.TestPreviews {
		err = akkeris.CreatePreviewHooks(diagnosticspec.App + "-" + diagnosticspec.Space)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

func UpdateDiagnostic(req *http.Request, diagnosticspec structs.DiagnosticSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
		return
	}

	err := updateDiagnostic(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}

	if diagnosticspec.TestPreviews {
		err = akkeris.CreatePreviewHooks(diagnosticspec.App + "-" + diagnosticspec.Space)
		if err != nil {
			fmt.Println(err)
		}
	} else if diagnosticspec.TestPreviews {
		err = akkeris.DeletePreviewHooks(diagnosticspec.App + "-" + diagnosticspec.Space)
		if err != nil {
			fmt.Println(err)
		}
	}

	dbstore.AddDiagnosticUpdateAudit(req, diagnosticspec)
	r.JSON(200, map[string]interface{}{"status": "updated"})
}

func updateDiagnostic(diagnosticspec structs.DiagnosticSpec) (e error) {
	err := akkeris.UpdateService(diagnosticspec)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}

func GetDiagnosticsList(req *http.Request, params martini.Params, r render.Render) {
	simple := req.URL.Query().Get("simple")

	diagnostics, err := getDiagnosticsList(simple)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	r.JSON(200, diagnostics)

}

func getDiagnosticsList(simple string) (d []structs.DiagnosticSpec, e error) {
	var diagnostics []structs.DiagnosticSpec
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return diagnostics, dberr
	}
	defer db.Close()
	stmt, err := db.Prepare("select id, space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,''), coalesce(testpreviews,null,false), coalesce(ispreview,null,false), coalesce(webhookurls,null,'') from diagnostics order by app, space")
	if err != nil {
		fmt.Println(err)
		return diagnostics, err
	}

	var did string
	var dspace string
	var dapp string
	var daction string
	var dresult string
	var djob string
	var djobspace string
	var dimage string
	var dpipelinename string
	var dtransitionfrom string
	var dtransitionto string
	var dtimeout int
	var dstartdelay int
	var dslackchannel string
	var dcommand string
	var dtestpreviews bool
	var dispreview bool
	var dwebhookurls string

	defer stmt.Close()
	rows, err := stmt.Query()
	for rows.Next() {
		err := rows.Scan(&did, &dspace, &dapp, &daction, &dresult, &djob, &djobspace, &dimage, &dpipelinename, &dtransitionfrom, &dtransitionto, &dtimeout, &dstartdelay, &dslackchannel, &dcommand, &dtestpreviews, &dispreview, &dwebhookurls)
		if err != nil {
			fmt.Println(err)
			return diagnostics, err
		}
		var diagnostic structs.DiagnosticSpec
		diagnostic.ID = did
		diagnostic.Space = dspace
		diagnostic.App = dapp
		diagnostic.Action = daction
		diagnostic.Result = dresult
		diagnostic.Job = djob
		diagnostic.JobSpace = djobspace
		diagnostic.Image = dimage
		diagnostic.PipelineName = dpipelinename
		diagnostic.TransitionFrom = dtransitionfrom
		diagnostic.TransitionTo = dtransitionto
		diagnostic.Timeout = dtimeout
		diagnostic.Startdelay = dstartdelay
		diagnostic.Slackchannel = dslackchannel
		diagnostic.Command = dcommand
		diagnostic.TestPreviews = dtestpreviews
		diagnostic.IsPreview = dispreview
		diagnostic.WebhookURLs = dwebhookurls
		runiduuid, _ := uuid.NewV4()
		runid := runiduuid.String()
		utils.PrintDebug(runid)
		diagnostic.RunID = runid
		if !(simple == "true") {
			envvars, _ := akkeris.GetVars(djob, djobspace)
			diagnostic.Env = envvars
		}
		diagnostics = append(diagnostics, diagnostic)

	}

	db.Close()

	return diagnostics, nil

}

func Rerun(req *http.Request, params martini.Params, r render.Render) {
	qs := req.URL.Query()
	space, app, action, result, releaseid, buildid := qs.Get("space"), qs.Get("app"), qs.Get("action"), qs.Get("result"), qs.Get("releaseid"), qs.Get("buildid")

	utils.PrintDebug(space)
	utils.PrintDebug(app)
	utils.PrintDebug(action)
	utils.PrintDebug(result)
	utils.PrintDebug(buildid)
	utils.PrintDebug(releaseid)
	err := rerun(space, app, action, result, buildid, releaseid)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	r.JSON(200, map[string]interface{}{"status": "rerunning"})

}
func rerun(space string, app string, action string, result string, buildid string, releaseid string) (e error) {
	diagnosticslist, err := GetDiagnostics(space, app, action, result)
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, element := range diagnosticslist {
		element.BuildID = buildid
		version, err := akkeris.GetVersion(element.App, element.Space, element.BuildID)
		if err != nil {
			fmt.Println(err)
			return err
		}
		utils.PrintDebug(version)
		var commitauthor string
		var commitmessage string
		if version != "" {
			element.GithubVersion = version
			commitauthor, commitmessage, err = githubapi.GetCommitAuthor(version)
			if err != nil {
				fmt.Println(err)
			}
			utils.PrintDebug(commitauthor)
		} else {
			commitauthor = "none"
			commitmessage = "none"
		}
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		element.ReleaseID = releaseid
		if element.ReleaseID == "" {
			utils.PrintDebug("release id not received.  Getting most recent")
			element.ReleaseID = dbstore.GetMostRecentReleaseID(element)
		}
		if element.ReleaseID == "" {
			utils.PrintDebug("release id not available in database.  Getting from controller")
			element.ReleaseID = akkeris.GetMostRecentReleaseID(element)
		}
		utils.PrintDebug("RELEASE ID : " + element.ReleaseID)
		RunDiagnostic(element, false, structs.Cronjob{})
	}
	return nil
}

func GetDiagnosticByNameOrID(params martini.Params, r render.Render) {
	var diagnostic structs.DiagnosticSpec

	provided := params["provided"]
	diagnostic, err := dbstore.FindDiagnostic(provided)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	envvars := diagnostic.Env
	var newenvvars []structs.EnvironmentVariable
	protectedspace, err := akkeris.IsProtectedSpace(diagnostic.Space)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	for _, element := range envvars {
		// System environment variables
		if (element.Name == "AKKERIS_CI") || (element.Name == "CI") {
			continue
		}
		if (strings.HasPrefix(element.Name, "TAAS_")) || (strings.HasPrefix(element.Name, "DIAGNOSTIC_")) {
			continue
		}

		// Redact protected variables
		if protectedspace && ((strings.Contains(element.Name, "SECRET")) || (strings.Contains(element.Name, "PASSWORD")) || (strings.Contains(element.Name, "TOKEN")) || (strings.Contains(element.Name, "KEY"))) {
			var newvar structs.EnvironmentVariable
			newvar.Name = element.Name
			newvar.Value = "[redacted]"
			newenvvars = append(newenvvars, newvar)
		} else {
			newenvvars = append(newenvvars, element)
		}
	}

	diagnostic.Env = newenvvars
	r.JSON(200, diagnostic)

}

func BindDiagnosticSecret(params martini.Params, r render.Render) {
	provided := params["provided"]
	spec := params["_1"]
	if spec == "undefined" {
		r.JSON(500, map[string]interface{}{"response": "invalid request"})
		return
	}
	utils.PrintDebug(provided)
	utils.PrintDebug(spec)
	diagnostic, err := dbstore.FindDiagnostic(provided)

	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	utils.PrintDebug(diagnostic.Job)
	utils.PrintDebug(diagnostic.JobSpace)
	var bind structs.Bindspec
	bind.App = diagnostic.Job
	bind.Space = diagnostic.JobSpace
	bind.Bindtype = strings.Split(spec, ":")[0]
	bind.Bindname = strings.Split(spec, ":")[1]
	p, err := json.Marshal(bind)
	if err != nil {
		fmt.Println(err)
	}
	if bind.Bindtype != "vault" {
		r.JSON(500, map[string]interface{}{"response": "can only bind vault"})
		return
	}
	//	if strings.Contains(bind.Bindname, "/prod/") {
	//		r.JSON(500, map[string]interface{}{"response": "can't bind prod"})
	//		return
	//	}
	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1/space/"+diagnostic.JobSpace+"/app/"+diagnostic.Job+"/bind", bytes.NewBuffer(p))
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
	if err != nil {
		fmt.Println(err)
	}
	utils.PrintDebug(string(bodybytes))
	r.JSON(200, map[string]interface{}{"response": "secret added"})
}

func UnbindDiagnosticSecret(params martini.Params, r render.Render) {
	provided := params["provided"]
	spec := params["_1"]
	if spec == "undefined" {
		r.JSON(500, map[string]interface{}{"response": "invalid request"})
		return
	}
	utils.PrintDebug(provided)
	utils.PrintDebug(spec)
	diagnostic, err := dbstore.FindDiagnostic(provided)

	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	utils.PrintDebug(diagnostic.Job)
	utils.PrintDebug(diagnostic.JobSpace)
	var bind structs.Bindspec
	bind.App = diagnostic.Job
	bind.Space = diagnostic.JobSpace
	bind.Bindtype = strings.Split(spec, ":")[0]
	bind.Bindname = strings.Split(spec, ":")[1]
	if bind.Bindtype != "vault" {
		r.JSON(500, map[string]interface{}{"response": "can only bind vault"})
		return
	}
	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("DELETE", akkerisapiurl+"/v1/space/"+diagnostic.JobSpace+"/app/"+diagnostic.Job+"/bind/"+bind.Bindtype+":"+bind.Bindname, nil)
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-type", "application/json")

	requestDump, _ := httputil.DumpRequest(req, false)
	utils.PrintDebug(string(requestDump))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	utils.PrintDebug(string(bodybytes))
	r.JSON(200, map[string]interface{}{"response": "secret removed"})

}

func isUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

func Octhc(params martini.Params, r render.Render) {
	r.Text(200, "overall_status=good")
}

func IsValidTest(test string) (v bool, e error) {
	var isvalid bool
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return false, dberr
	}
	defer db.Close()
	stmt, err := db.Prepare("select id from diagnostics where job||'-'||jobspace = $1")
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(test)
	defer rows.Close()

	if err != nil {
		return false, err
	}
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if isUUID(id) {
			isvalid = true
		} else {
			isvalid = false
		}
	}
	db.Close()
	return isvalid, nil
}

func SetConfig(req *http.Request, params martini.Params, varspec structs.Varspec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
		return
	}

	diagnostic, err := dbstore.FindDiagnostic(params["provided"])
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}

	if diagnostic.ID != "" {
		utils.PrintDebug("valid test")
	} else {
		r.JSON(400, map[string]interface{}{"response": "bad request - test does not exist"})
		return
	}

	varspec.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	utils.PrintDebug(varspec.Setname + ": " + varspec.Varname + "=" + varspec.Varvalue)
	existing, err := akkeris.GetVars(diagnostic.Job, diagnostic.JobSpace)
	var exists bool
	exists = false
	for _, element := range existing {
		if element.Name == varspec.Varname {
			exists = true
			break
		}
	}
	if exists {
		//update
		utils.PrintDebug("Updating")
		err = akkeris.UpdateVar(varspec)
		if err != nil {
			fmt.Println(err)
			r.JSON(200, map[string]interface{}{"response": err.Error()})
			return
		}
	} else {
		utils.PrintDebug("Adding")
		err = akkeris.AddVar(varspec)
		if err != nil {
			fmt.Println(err)
			r.JSON(200, map[string]interface{}{"response": err.Error()})
			return
		}
	}
	dbstore.AddConfigSetAudit(req, diagnostic.ID, varspec)
	r.JSON(200, map[string]interface{}{"response": "config variable set"})

}

func UnsetConfig(req *http.Request, params martini.Params, r render.Render) {
	varname := params["varname"]
	provided := params["provided"]
	utils.PrintDebug(varname)
	utils.PrintDebug(provided)
	diagnostic, err := dbstore.FindDiagnostic(params["provided"])
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(400, map[string]interface{}{"response": "bad request - test does not exist"})
		return
	}
	err = akkeris.DeleteVar(diagnostic, varname)
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	dbstore.AddConfigUnsetAudit(req, diagnostic.ID, varname)
	r.JSON(200, map[string]interface{}{"response": "config variable unset"})

}

func CreateHooks(params martini.Params, r render.Render) {
	var diagnostic structs.DiagnosticSpec
	provided := params["provided"]
	diagnostic, err := dbstore.FindDiagnostic(provided)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}

	err = jobs.CreateHooks(diagnostic.App + "-" + diagnostic.Space)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}

	r.JSON(200, map[string]interface{}{"status": "hooks added"})
}

func describePodAndUploadToS3(space string, name string, runid string) (p structs.TemplatePod, e error) {
	var templatepod structs.TemplatePod
	object, err := akkeris.DescribePod(space, name)
	if err != nil {
		fmt.Println(err)
		return templatepod, err
	}
	templatepod.Name = object.Metadata.Name
	templatepod.Space = object.Metadata.Namespace
	templatepod.Node = object.Spec.NodeName
	templatepod.StartTime = object.Status.StartTime
	templatepod.Status = object.Status.Phase
	templatepod.Containers = object.Spec.Containers
	templatepod.Conditions = object.Status.Conditions
	templatepod.Events = object.Events.Items

	describetemplate := `
Name:               {{ .Name }}
Namespace:          {{ .Space }}
Node:               {{ .Node }}
Start Time:         {{ .StartTime }}
Status:             {{ .Status }}

Containers:
-----------{{range .Containers}}
  {{ .Name }}:
    Image:          {{ .Image }}
{{end}}
Conditions:
-----------{{range .Conditions}}
  Type: {{ .Type }}
  Status: {{ .Status }}
  Reason: {{ .Reason }}
  Message: {{ .Message }}
{{end}}

Events:
-------{{range .Events}}
  Time: {{ .LastTimestamp }}
  Type: {{ .Type }}
  Reason: {{ .Reason }}
  Message: {{ .Message }}
{{end}}
`
	var t *template.Template
	t = template.Must(template.New("desribe").Parse(describetemplate))
	var b bytes.Buffer
	wr := bufio.NewWriter(&b)
	err = t.Execute(wr, templatepod)
	if err != nil {
		utils.PrintDebug(err.Error())
	}
	wr.Flush()

	artifacts.UploadToS3(string(b.Bytes()), "text/plain", runid)
	return templatepod, nil
}

// Return a list of all currently running jobs
func GetCurrentRuns(req *http.Request, params martini.Params, r render.Render) {
	type AllPendingRuns struct {
		CurrentRuns     []structs.PendingRun     `json:"current_runs"`
		CurrentCronRuns []structs.PendingCronRun `json:"current_cron_runs"`
	}

	runs, err := dbstore.GetCurrentRuns()
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}

	cronRuns, err := dbstore.GetCurrentCronRuns()
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}

	var resp AllPendingRuns
	resp.CurrentCronRuns = cronRuns
	resp.CurrentRuns = runs

	r.JSON(200, resp)
}
