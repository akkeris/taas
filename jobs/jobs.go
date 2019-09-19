package jobs

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"

	vault "github.com/akkeris/vault-client"
	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
)

func UpdateService(diagnosticspec structs.DiagnosticSpec) (e error) {
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}
	defer db.Close()
	fmt.Println(diagnosticspec.Slackchannel)
	stmt, err := db.Prepare("UPDATE diagnostics set image=$1,pipelinename=$2,transitionfrom=$3,transitionto=$4,timeout=$5,startdelay=$6,slackchannel=$7,command=$8,testpreviews=$9 where job=$10 and jobspace=$11")
	if err != nil {
		fmt.Println(err)
		return err
	}
	res, err := stmt.Exec(
		diagnosticspec.Image, diagnosticspec.PipelineName, diagnosticspec.TransitionFrom,
		diagnosticspec.TransitionTo, diagnosticspec.Timeout, diagnosticspec.Startdelay,
		diagnosticspec.Slackchannel, diagnosticspec.Command, diagnosticspec.TestPreviews,
		diagnosticspec.Job, diagnosticspec.JobSpace,
	)
	if err != nil {
		fmt.Println(err)
		return err
	}
	rowCnt, err := res.RowsAffected()

	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(rowCnt)
	return nil
}

func CreateService(diagnosticspec structs.DiagnosticSpec) (e error) {
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}
	defer db.Close()
	diagnosticspec.TransitionTo = strings.Replace(diagnosticspec.TransitionTo, " ", "", -1)

	var id string
	inserterr := db.QueryRow(
		"INSERT INTO diagnostics(id, space, app, action, result, job, jobspace,image,pipelinename,transitionfrom,transitionto,timeout,startdelay,slackchannel,command,testpreviews,ispreview) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) returning id;",
		diagnosticspec.ID, diagnosticspec.Space, diagnosticspec.App, diagnosticspec.Action,
		diagnosticspec.Result, diagnosticspec.Job, diagnosticspec.JobSpace,
		diagnosticspec.Image, diagnosticspec.PipelineName, diagnosticspec.TransitionFrom,
		diagnosticspec.TransitionTo, diagnosticspec.Timeout, diagnosticspec.Startdelay,
		diagnosticspec.Slackchannel, diagnosticspec.Command, diagnosticspec.TestPreviews, diagnosticspec.IsPreview,
	).Scan(&id)
	if inserterr != nil {
		return inserterr
	}

	return nil
}

func CreateBind(diagnosticspec structs.DiagnosticSpec) (e error) {
	type Bindspec struct {
		App      string `json:"appname"`
		Space    string `json:"space"`
		Bindtype string `json:"bindtype"`
		Bindname string `json:"bindname"`
	}
	var bindspec Bindspec
	bindspec.App = diagnosticspec.Job
	bindspec.Space = diagnosticspec.JobSpace
	bindspec.Bindtype = "config"
	bindspec.Bindname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"

	p, err := json.Marshal(bindspec)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1/space/"+bindspec.Space+"/app/"+bindspec.App+"/bind", bytes.NewBuffer(p))
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

func CreateConfigSet(diagnosticspec structs.DiagnosticSpec) (e error) {

	type Setspec struct {
		Setname string `json:"name"`
		Settype string `json:"type"`
	}

	var setspec Setspec
	setspec.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	setspec.Settype = "diagnostic"

	p, err := json.Marshal(setspec)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1/config/set", bytes.NewBuffer(p))
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

func CreateVariables(diagnosticspec structs.DiagnosticSpec) (e error) {

	var vars []structs.Varspec

	for _, element := range diagnosticspec.Env {
		var varspec structs.Varspec
		varspec.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
		varspec.Varname = element.Name
		varspec.Varvalue = element.Value
		vars = append(vars, varspec)
	}
	var logvar structs.Varspec
	logvar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	logvar.Varname = "DIAGNOSTIC_LOG_ENDPOINT"
	logvar.Varvalue = os.Getenv("LOG_URL") + "/jobspace/" + diagnosticspec.JobSpace + "/job/" + diagnosticspec.Job + "/logs"
	vars = append(vars, logvar)

	var jobvar structs.Varspec
	jobvar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	jobvar.Varname = "DIAGNOSTIC_JOB_NAME"
	jobvar.Varvalue = diagnosticspec.Job
	vars = append(vars, jobvar)

	var spacevar structs.Varspec
	spacevar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	spacevar.Varname = "DIAGNOSTIC_JOB_SPACE"
	spacevar.Varvalue = diagnosticspec.JobSpace
	vars = append(vars, spacevar)

	var appspacevar structs.Varspec
	appspacevar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	appspacevar.Varname = "DIAGNOSTIC_APP_SPACE"
	appspacevar.Varvalue = diagnosticspec.Space
	vars = append(vars, appspacevar)

	var appvar structs.Varspec
	appvar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	appvar.Varname = "DIAGNOSTIC_APP"
	appvar.Varvalue = diagnosticspec.App
	vars = append(vars, appvar)

	var runidvar structs.Varspec
	runiduuid, _ := uuid.NewV4()
	runid := runiduuid.String()
	runidvar.Setname = diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	runidvar.Varname = "DIAGNOSTIC_RUNID"
	runidvar.Varvalue = runid
	vars = append(vars, runidvar)

	p, err := json.Marshal(vars)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1/config/set/configvar", bytes.NewBuffer(p))
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

func UpdateVar(vartoadd structs.Varspec) error {
	if vartoadd.Varvalue == "[redacted]" {
		return errors.New("unable to set value of " + vartoadd.Varname + " to " + vartoadd.Varvalue)
	}
	p, err := json.Marshal(vartoadd)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("PATCH", akkerisapiurl+"/v1/config/set/configvar", bytes.NewBuffer(p))
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

func AddVar(vartoadd structs.Varspec) error {
	if vartoadd.Varvalue == "[redacted]" {
		return errors.New("unable to set value of " + vartoadd.Varname + " to " + vartoadd.Varvalue)
	}
	var vars []structs.Varspec
	vars = append(vars, vartoadd)

	p, err := json.Marshal(vars)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1/config/set/configvar", bytes.NewBuffer(p))
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

func DeleteVar(diagnosticspec structs.DiagnosticSpec, varname string) error {
	setname := diagnosticspec.Job + "-" + diagnosticspec.JobSpace + "-cs"
	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("DELETE", akkerisapiurl+"/v1/config/set/"+setname+"/configvar/"+varname, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	fmt.Println(req)
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

func DeleteService(diagnostic structs.DiagnosticSpec) (e error) {

	uri := os.Getenv("DIAGNOSTICDB")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("delete from diagnostics where id=$1")
	if err != nil {
		fmt.Println(err)
		return err
	}
	res, err := stmt.Exec(diagnostic.ID)
	if err != nil {
		fmt.Println(err)
		return err
	}
	affect, err := res.RowsAffected()
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(affect)

	db.Close()

	return nil
}

func DeleteBind(diagnostic structs.DiagnosticSpec) (e error) {

	bindname := diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("DELETE", akkerisapiurl+"/v1/space/"+diagnostic.JobSpace+"/app/"+diagnostic.Job+"/bind/config:"+bindname, nil)
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

func DeleteSet(diagnostic structs.DiagnosticSpec) (e error) {

	bindname := diagnostic.Job + "-" + diagnostic.JobSpace + "-cs"

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("DELETE", akkerisapiurl+"/v1/config/set/"+bindname, nil)
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

func DeleteJob(diagnostic structs.DiagnosticSpec) (e error) {

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("DELETE", akkerisapiurl+"/v1beta1/space/"+diagnostic.JobSpace+"/jobs/"+diagnostic.Job, nil)
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

func toAppSpace(full string) (s string, a string) {
	parts := strings.Split(full, "-")
	app := parts[0]
	rest := parts[1:]
	return app, strings.Join(rest, "-")
}

func GetCurrentImage(app string) (i string) {
	app, space := toAppSpace(app)
	req, err := http.NewRequest("GET", os.Getenv("AKKERIS_API_URL")+"/v1/space/"+space+"/app/"+app, nil)
	if err != nil {
		fmt.Println(err)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	var appinfo structs.AppInfo
	err = json.Unmarshal(bodybytes, &appinfo)
	if err != nil {
		fmt.Println(err)
	}
	return appinfo.Image
}

// CreateHooks - Check presence of build/release hooks on an app and add them if needed
func CreateHooks(appspace string) (e error) {
	svcurl := os.Getenv("TAAS_SVC_URL")
	var failedHooks []string

	hooks, err := GetHooks(appspace)
	if err != nil {
		return err
	}

	needsBuild := true
	needsRelease := true
	for _, hook := range hooks {
		if needsBuild {
			needsBuild = !strings.Contains(hook.URL, svcurl+"/v1/buildhook")
		}
		if needsRelease {
			needsRelease = !strings.Contains(hook.URL, svcurl+"/v1/releasehook")
		}
	}

	if needsBuild {
		err := CreateHook(true, []string{"build"}, svcurl+"/v1/buildhook", "merpderp", appspace)
		if err != nil {
			fmt.Println("Error creating build hook")
			fmt.Println(err)
			failedHooks = append(failedHooks, "build")
		}
	}

	if needsRelease {
		err := CreateHook(true, []string{"release"}, svcurl+"/v1/releasehook", "merpderp", appspace)
		if err != nil {
			fmt.Println("Error creating release hook")
			fmt.Println(err)
			failedHooks = append(failedHooks, "release")
		}
	}

	if len(failedHooks) != 0 {
		return errors.New("One or more hooks failed to create: " + strings.Join(failedHooks, ","))
	}

	fmt.Println("All hooks present!")
	return nil
}

func CreatePreviewHooks(appspace string) (e error) {
	fmt.Println("Creating preview hooks...")
	svcurl := os.Getenv("TAAS_SVC_URL")
	var failedHooks []string

	hooks, err := GetHooks(appspace)
	if err != nil {
		return err
	}

	needsReleased := true
	needsCreated := true
	for _, hook := range hooks {
		if needsReleased {
			needsReleased = !strings.Contains(hook.URL, svcurl+"/v1/previewreleasedhook")
		}
		if needsCreated {
			needsCreated = !strings.Contains(hook.URL, svcurl+"/v1/previewcreatedhook")
		}
	}

	if needsReleased {
		err := CreateHook(true, []string{"preview-released"}, svcurl+"/v1/previewreleasedhook", "merpderp", appspace)
		if err != nil {
			fmt.Println("Error creating preview released hook")
			fmt.Println(err)
			failedHooks = append(failedHooks, "preview-released")
		}
	}

	if needsCreated {
		err := CreateHook(true, []string{"preview"}, svcurl+"/v1/previewcreatedhook", "merpderp", appspace)
		if err != nil {
			fmt.Println("Error creating preview created hook")
			fmt.Println(err)
			failedHooks = append(failedHooks, "preview")
		}
	}

	if len(failedHooks) != 0 {
		return errors.New("One or more hooks failed to create: " + strings.Join(failedHooks, ","))
	}

	fmt.Println("All preview hooks present!")
	return nil
}

func CreateHook(active bool, events []string, url string, secret string, app string) (e error) {
	var controllerurl = os.Getenv("APP_CONTROLLER_URL")

	var hook structs.AppHook
	hook.Active = active
	hook.Events = events
	hook.URL = url
	hook.Secret = secret

	h, err := json.Marshal(hook)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", controllerurl+"/apps/"+app+"/hooks", bytes.NewBuffer(h))
	if err != nil {
		return err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", vault.GetField(os.Getenv("APP_CONTROLLER_AUTH_SECRET"), "authorization"))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(bodybytes))
	return nil
}

func DeletePreviewHooks(appspace string) (e error) {
	fmt.Println("Deleting preview hooks...")
	svcurl := os.Getenv("TAAS_SVC_URL")

	hooks, err := GetHooks(appspace)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		if strings.Contains(hook.URL, svcurl+"/v1/previewreleasedhook") || strings.Contains(hook.URL, svcurl+"/v1/previewcreatedhook") {
			err := DeleteHook(appspace, hook.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
