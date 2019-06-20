package diagnostics

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	dbstore "taas/dbstore"
	diagnosticlogs "taas/diagnosticlogs"
	githubapi "taas/githubapi"
	akkeris "taas/jobs"
	jobs "taas/jobs"
	notifications "taas/notifications"
	pipelines "taas/pipelines"
	structs "taas/structs"
	"time"

	"github.com/go-martini/martini"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
)

func RunDiagnostic(diagnostic structs.DiagnosticSpec) (e error) {

	// may need to inject the run id into the config set at this point so that it is available to internal code if it will send logs

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

	go check(diagnostic)
	return nil
}

func check(diagnostic structs.DiagnosticSpec) {

	fmt.Println("Start Delay Set to : " + strconv.Itoa(diagnostic.Startdelay))
	time.Sleep(time.Second * time.Duration(diagnostic.Startdelay))

	var oneoff structs.OneOffSpec
	oneoff.Space = diagnostic.JobSpace
	oneoff.Podname = strings.ToLower(diagnostic.Job)
	if strings.HasPrefix(diagnostic.Image, "akkeris://") {
		imageappname := strings.Replace(diagnostic.Image, "akkeris://", "", -1)
		currentimage := akkeris.GetCurrentImage(imageappname)
		oneoff.Image = currentimage
		diagnostic.Image = currentimage
	} else {
		fmt.Println("assuming docker image url")
		oneoff.Image = diagnostic.Image
	}
	oneoff.Command = diagnostic.Command
	fetched, err := akkeris.GetVars(diagnostic.Job, diagnostic.JobSpace)
	if err != nil {
		fmt.Println(err)
	}
	oneoff.Env = fetched

	akkeris.Deletepod(oneoff.Space, oneoff.Podname)
	time.Sleep(time.Second * 5)
	akkeris.Startpod(oneoff)

	time.Sleep(time.Second * 3)

	starttime := time.Now().UTC()
	endtime := time.Now().UTC()
	var instance string
	var overallstatus string
	overallstatus = "timedout"
	var i float64
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
		fmt.Println(string(bodybytes))
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
			fmt.Println("JOB FAILED")
			overallstatus = "failed"
			endtime = time.Now().UTC()
			for _, element := range status {
				if element.Reason == "Error" || element.Phase == "Running/terminated" || element.Phase == "Failed/terminated" {
					fmt.Println(element.Instanceid)
					instance = element.Instanceid
					fmt.Println(element.Phase)
					fmt.Println(element.Reason)
				}
			}
			break
		}
		instance = status[0].Instanceid
		if status[0].Phase == "Succeeded/terminated" && status[0].Reason == "Completed" {
			fmt.Println("JOB FINISHED OK")
			overallstatus = "success"
			endtime = time.Now().UTC()
			break
		}

		if status[0].Phase == "Running/terminated" && status[0].Reason == "Error" {
			fmt.Println("JOB FAILED")
			overallstatus = "failed"
			endtime = time.Now().UTC()
			break
		}

		if status[0].Phase == "Running/waiting" && status[0].Reason == "CrashLoopBackOff" {
			fmt.Println("JOB FAILED")
			overallstatus = "failed"
			endtime = time.Now().UTC()
			break
		}
		if status[0].Phase == "Failed/terminated" && status[0].Reason == "Error" {
			fmt.Println("JOB FAILED")
			overallstatus = "failed"
			endtime = time.Now().UTC()
			break
		}
	}
	fmt.Println("finishing....")
	logs, err := jobs.GetTestLogs(diagnostic.JobSpace, diagnostic.Job, instance)
	if err != nil {
		fmt.Println(err)
	}
	err = akkeris.ScaleJob(diagnostic.JobSpace, diagnostic.Job, 0, 0)

	if err != nil {
		fmt.Println(err)
	}
	diagnostic.OverallStatus = overallstatus
	var loglines structs.LogLines
	loglines.Logs = logs
	diagnosticlogs.WriteLogES(diagnostic, loglines)
	err = dbstore.StoreRun(diagnostic)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("done")
	fmt.Println(overallstatus)
	notifications.PostToSlack(diagnostic, overallstatus)
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

	var action structs.ActionSpec
	action.Name = diagnostic.Job + "-" + diagnostic.JobSpace
	action.Status = overallstatus
	action.Messages = logs
	var actions []structs.ActionSpec
	actions = append(actions, action)
	step.Actions = actions

	var steps []structs.StepSpec
	steps = append(steps, step)
	result.Payload.Steps = steps
	if err != nil {
		fmt.Println(err)
	}
	if overallstatus == "success" && diagnostic.PipelineName != "manual" {
		transitionfrom := diagnostic.TransitionFrom
		transitionto := diagnostic.TransitionTo
		transitiontoa := strings.Split(transitionto, ",")
		fmt.Println("transition from: " + transitionfrom)
		fmt.Println("transition to: " + transitionto)

		var fromappid string
		var toappids []string
		var pipelineid string
		pipeline, err := pipelines.GetPipeline(diagnostic.PipelineName)
		if err != nil {
			fmt.Println(err)
		}
		for _, element := range pipeline {
			fmt.Println(element.Stage)
			fmt.Println(element.App.Name)
			if element.Stage+":"+element.App.Name == transitionfrom {
				fromappid = element.App.ID
				pipelineid = element.Pipeline.ID
			}
			for _, trelement := range transitiontoa {
				if element.Stage+":"+element.App.Name == trelement {
					fmt.Println("setting app id for target to " + element.App.ID)
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
		err = pipelines.PromoteApp(promotion)

		if err != nil {
			fmt.Println(err)
		}
	}
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
	fmt.Println(string(bodybytes))

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
	stmt, err := db.Prepare("select id, space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay,slackchannel,coalesce(command,null,'') from diagnostics where space = $1 and app = $2 and action = $3 and result=$4")
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

	defer stmt.Close()
	rows, err := stmt.Query(space, app, action, result)
	for rows.Next() {
		err := rows.Scan(&did, &dspace, &dapp, &daction, &dresult, &djob, &djobspace, &dimage, &dpipelinename, &dtransitionfrom, &dtransitionto, &dtimeout, &dstartdelay, &dslackchannel, &dcommand)
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
		runiduuid, _ := uuid.NewV4()
		runid := runiduuid.String()
		fmt.Println(runid)
		diagnostic.RunID = runid
		diagnostics = append(diagnostics, diagnostic)
	}

	db.Close()

	return diagnostics, nil

}

func DeleteDiagnostic(params martini.Params, r render.Render) {
	diagnostic, err := getDiagnosticByNameOrID(params["provided"])
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}

	err = deleteDiagnostic(diagnostic)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return

	}

	r.JSON(200, map[string]interface{}{"status": "deleted"})

}

func deleteDiagnostic(diagnostic structs.DiagnosticSpec) (e error) {

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

func CreateDiagnostic(diagnosticspec structs.DiagnosticSpec, berr binding.Errors, r render.Render) {

	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
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
	err = createDiagnostic(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return

	}

	r.JSON(200, map[string]interface{}{"status": "created"})

}

func createDiagnostic(diagnosticspec structs.DiagnosticSpec) (e error) {
	err := akkeris.CreateJob(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = akkeris.CreateConfigSet(diagnosticspec)
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

	return nil
}

func UpdateDiagnostic(diagnosticspec structs.DiagnosticSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
	}
	fmt.Println(diagnosticspec)
	err := updateDiagnostic(diagnosticspec)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return

	}

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
	stmt, err := db.Prepare("select id, space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,'') from diagnostics order by app, space")
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

	defer stmt.Close()
	rows, err := stmt.Query()
	for rows.Next() {
		err := rows.Scan(&did, &dspace, &dapp, &daction, &dresult, &djob, &djobspace, &dimage, &dpipelinename, &dtransitionfrom, &dtransitionto, &dtimeout, &dstartdelay, &dslackchannel, &dcommand)
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
		runiduuid, _ := uuid.NewV4()
		runid := runiduuid.String()
		fmt.Println(runid)
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
	space, app, action, result, buildid := qs.Get("space"), qs.Get("app"), qs.Get("action"), qs.Get("result"), qs.Get("buildid")

	fmt.Println(space)
	fmt.Println(app)
	fmt.Println(action)
	fmt.Println(result)
	fmt.Println(buildid)
	err := rerun(space, app, action, result, buildid)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
	}
	r.JSON(200, map[string]interface{}{"status": "rerunning"})

}
func rerun(space string, app string, action string, result string, buildid string) (e error) {
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
		fmt.Println(version)
		var commitauthor string
		var commitmessage string
		if version != "" {
			element.GithubVersion = version
			commitauthor, commitmessage, err = githubapi.GetCommitAuthor(version)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(commitauthor)
		} else {
			commitauthor = "none"
			commitmessage = "none"
		}
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		RunDiagnostic(element)
	}
	return nil
}

func GetDiagnosticByNameOrID(params martini.Params, r render.Render) {
	var diagnostic structs.DiagnosticSpec

	provided := params["provided"]
	diagnostic, err := getDiagnosticByNameOrID(provided)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
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
		if (strings.HasPrefix(element.Name, "TAAS_")) || (strings.HasPrefix(element.Name, "DIAGNOSTIC_")) {
			continue
		}

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

func getDiagnosticByNameOrID(provided string) (d structs.DiagnosticSpec, e error) {
	var diagnostic structs.DiagnosticSpec
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return diagnostic, dberr
	}
	defer db.Close()
	var selectstring string
	if !isUUID(provided) {
		selectstring = "select id,  space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,'') from diagnostics where job||'-'||jobspace = $1"
	} else {
		selectstring = "select id,  space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,'') from diagnostics where id = $1"
	}
	stmt, err := db.Prepare(selectstring)
	if err != nil {
		fmt.Println(err)
		return diagnostic, err
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

	defer stmt.Close()
	rows, err := stmt.Query(provided)
	for rows.Next() {
		err := rows.Scan(&did, &dspace, &dapp, &daction, &dresult, &djob, &djobspace, &dimage, &dpipelinename, &dtransitionfrom, &dtransitionto, &dtimeout, &dstartdelay, &dslackchannel, &dcommand)
		if err != nil {
			fmt.Println(err)
			return diagnostic, err
		}
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
		//runiduuid, _ := uuid.NewV4()
		//runid := runiduuid.String()
		//fmt.Println(runid)
		//diagnostic.RunID = runid
		envvars, _ := akkeris.GetVars(djob, djobspace)
		diagnostic.Env = envvars
	}

	db.Close()

	return diagnostic, nil

}

func BindDiagnosticSecret(params martini.Params, r render.Render) {
	provided := params["provided"]
	spec := params["_1"]
	if spec == "undefined" {
		r.JSON(500, map[string]interface{}{"response": "invalid request"})
		return
	}
	fmt.Println(provided)
	fmt.Println(spec)
	diagnostic, err := getDiagnosticByNameOrID(provided)

	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	fmt.Println(diagnostic.Job)
	fmt.Println(diagnostic.JobSpace)
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
	fmt.Println(string(bodybytes))
	r.JSON(200, map[string]interface{}{"response": "secret added"})
}

func UnbindDiagnosticSecret(params martini.Params, r render.Render) {
	provided := params["provided"]
	spec := params["_1"]
	if spec == "undefined" {
		r.JSON(500, map[string]interface{}{"response": "invalid request"})
		return
	}
	fmt.Println(provided)
	fmt.Println(spec)
	diagnostic, err := getDiagnosticByNameOrID(provided)

	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	fmt.Println(diagnostic.Job)
	fmt.Println(diagnostic.JobSpace)
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
	fmt.Println(req)
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
	fmt.Println(string(bodybytes))
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

func SetConfig(params martini.Params, varspec structs.Varspec, berr binding.Errors, r render.Render) {

	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
	}
	diagnostic, err := getDiagnosticByNameOrID(params["provided"])
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
	}
	if diagnostic.ID != "" {
		fmt.Println("valid test")
	} else {
		r.JSON(400, map[string]interface{}{"response": "bad request - test does not exist"})
	}
	varspec.Setname = diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"
	fmt.Println(varspec)
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
		fmt.Println("Updating")
		err = akkeris.UpdateVar(varspec)
		if err != nil {
			fmt.Println(err)
			r.JSON(200, map[string]interface{}{"response": err.Error()})
			return
		}
	} else {
		fmt.Println("Adding")
		err = akkeris.AddVar(varspec)
		if err != nil {
			fmt.Println(err)
			r.JSON(200, map[string]interface{}{"response": err.Error()})
			return
		}
	}
	r.JSON(200, map[string]interface{}{"response": "config variable set"})

}

func UnsetConfig(params martini.Params, r render.Render) {
	varname := params["varname"]
	provided := params["provided"]
	fmt.Println(varname)
	fmt.Println(provided)
	diagnostic, err := getDiagnosticByNameOrID(params["provided"])
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
	}
	if diagnostic.ID == "" {
		r.JSON(400, map[string]interface{}{"response": "bad request - test does not exist"})
	}
	err = akkeris.DeleteVar(diagnostic, varname)
	if err != nil {
		r.JSON(500, map[string]interface{}{"response": err})
	}
	r.JSON(200, map[string]interface{}{"response": "config variable unset"})

}
