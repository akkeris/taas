package jobs

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	structs "taas/structs"

	_ "github.com/lib/pq"
	"github.com/nu7hatch/gouuid"
)

func UpdateService(diagnosticspec structs.DiagnosticSpec) (e error) {
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}
	defer db.Close()
	fmt.Println(diagnosticspec.Slackchannel)
	stmt, err := db.Prepare("UPDATE diagnostics set job=$1, jobspace=$2,image=$3,pipelinename=$4,transitionfrom=$5,transitionto=$6,timeout=$7,startdelay=$8,slackchannel=$9 where app=$10 and space=$11 and action=$12 and result=$13")
	if err != nil {
		fmt.Println(err)
		return err
	}
	res, err := stmt.Exec(
		diagnosticspec.Job, diagnosticspec.JobSpace, diagnosticspec.Image,
		diagnosticspec.PipelineName, diagnosticspec.TransitionFrom,
		diagnosticspec.TransitionTo, diagnosticspec.Timeout, diagnosticspec.Startdelay,
		diagnosticspec.Slackchannel, diagnosticspec.App, diagnosticspec.Space,
		diagnosticspec.Action, diagnosticspec.Result,
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

	newappiduuid, _ := uuid.NewV4()
	newappid := newappiduuid.String()

	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}
	defer db.Close()
	diagnosticspec.TransitionTo = strings.Replace(diagnosticspec.TransitionTo, " ", "", -1)

	var id string
	inserterr := db.QueryRow(
		"INSERT INTO diagnostics(id, space, app, action, result, job, jobspace,image,pipelinename,transitionfrom,transitionto,timeout,startdelay,slackchannel) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) returning id;",
		newappid, diagnosticspec.Space, diagnosticspec.App, diagnosticspec.Action,
		diagnosticspec.Result, diagnosticspec.Job, diagnosticspec.JobSpace,
		diagnosticspec.Image, diagnosticspec.PipelineName, diagnosticspec.TransitionFrom,
		diagnosticspec.TransitionTo, diagnosticspec.Timeout, diagnosticspec.Startdelay,
		diagnosticspec.Slackchannel,
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

func CreateJob(diagnosticspec structs.DiagnosticSpec) (e error) {

	type JobReq struct {
		Name     string `json:"name"`  // required
		Space    string `json:"space"` // required
		CMD      string `json:"cmd,omitempty"`
		Schedule string `json:"schedule,omitempty"`
		Plan     string `json:"plan"`
	}

	var jobreq JobReq
	jobreq.Name = diagnosticspec.Job
	jobreq.Space = diagnosticspec.JobSpace
	jobreq.Plan = "standard-s"

	p, err := json.Marshal(jobreq)
	if err != nil {
		fmt.Println(err)
		return err
	}

	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	req, err := http.NewRequest("POST", akkerisapiurl+"/v1beta1/jobs", bytes.NewBuffer(p))
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


func toAppSpace(full string)(s string, a string){
      parts := strings.Split(full,"-")
      app := parts[0]
      rest := parts[1:]
      return app, strings.Join(rest,"-")
}

func GetCurrentImage(app string)(i string){
       app, space  := toAppSpace(app)
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

