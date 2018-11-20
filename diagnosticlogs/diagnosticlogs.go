package diagnosticlogs

import (
	"bytes"
	structs "taas/structs"
	//"github.com/nu7hatch/gouuid"
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

func GetLogs(space string, job string, instance string) (l []string, e error) {
	alamoapiurl := os.Getenv("ALAMO_API_URL")
	var lines []string
	req, err := http.NewRequest("GET", alamoapiurl+"/v1/space/"+space+"/app/"+job+"/instance/"+instance+"/log", nil)
	if err != nil {
		fmt.Println(err)
		return lines, err
	}
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return lines, err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return lines, err
	}

	var log struct {
		Logs string `json:"logs"`
	}
	err = json.Unmarshal(bodybytes, &log)
	if err != nil {
		fmt.Println(err)
		return lines, err
	}
	//        fmt.Println(log)
	lines = strings.Split(log.Logs, "\n")
	//       fmt.Println(lines)
	return lines, nil

}

func WriteLog(logs structs.LogLines, berr binding.Errors, params martini.Params, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
	}
	job := params["job"]
	InternalWriteLog(job, logs)
}

func InternalWriteLog(job string, logs structs.LogLines) {
	t := time.Now()
	stamp := t.Format(time.RFC3339)
	os.MkdirAll("./static/logs/"+job+"/", os.ModePerm)

	file, err := os.OpenFile("./static/logs/"+job+"/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	defer file.Close()

	for _, element := range logs.Logs {
		_, err = io.WriteString(file, stamp+" "+element+"\n")
		if err != nil {
			fmt.Println(err)
		}
	}

}

func WriteLogExtended(logs structs.LogLines, berr binding.Errors, params martini.Params, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
	}
	job := params["job"]
	jobspace := params["jobspace"]
	InternalWriteLogExtended(jobspace, job, logs)
}

func InternalWriteLogExtended(jobspace string, job string, logs structs.LogLines) {
	t := time.Now()
	stamp := t.Format(time.RFC3339)

	os.MkdirAll("./static/logs/jobspace/"+jobspace+"/job/"+job+"/", os.ModePerm)

	file, err := os.OpenFile("./static/logs/jobspace/"+jobspace+"/job/"+job+"/log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	for _, element := range logs.Logs {

		//		fmt.Println(stamp + " " + element)

		if err != nil {
			fmt.Println(err)
		}
		defer file.Close()
		_, err = io.WriteString(file, stamp+" "+element+"\n")
	}

}

func Logs(params martini.Params, r render.Render) {
	job := params["job"]
	fmt.Println(job)
	b, err := ioutil.ReadFile("./static/logs/" + job + "/log.txt")
	if err != nil {
		fmt.Print(err)
	}
	//fmt.Println(b)
	str := string(b)
	fmt.Println(str)
	r.Text(200, str)

}

func LogsExtended(params martini.Params, r render.Render) {
	job := params["job"]
	jobspace := params["jobspace"]
	fmt.Println(job)
	b, err := ioutil.ReadFile("./static/logs/jobspace/" + jobspace + "/job/" + job + "/log.txt")
	if err != nil {
		fmt.Print(err)
	}
	fmt.Println(b)
	str := string(b)
	//fmt.Println(str)
	r.Text(200, str)

}

func WriteLogES(diagnostic structs.DiagnosticSpec, logs structs.LogLines) {
	timestamp := int(int32(time.Now().Unix()))
	t := time.Now()
	hrtimestamp := t.Format(time.RFC3339)
	var eslogs structs.ESlogSpecIn
	eslogs.Job = diagnostic.Job
	eslogs.Jobspace = diagnostic.JobSpace
	eslogs.App = diagnostic.App
	eslogs.Space = diagnostic.Space
	eslogs.Testid = diagnostic.Job + "-" + diagnostic.JobSpace + "-" + diagnostic.App + "-" + diagnostic.Space
	eslogs.RunID = diagnostic.RunID
	eslogs.OverallStatus = diagnostic.OverallStatus
	eslogs.Timestamp = timestamp
	eslogs.Hrtimestamp = hrtimestamp
	eslogs.BuildID = diagnostic.BuildID
	eslogs.Organization = diagnostic.Organization
	eslogs.Logs = logs.Logs

	p, err := json.Marshal(eslogs)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("PUT", os.Getenv("ES_URL")+"/logs/run/"+diagnostic.RunID, bytes.NewBuffer(p))
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
	fmt.Println(string(bodybytes))
}

func GetLogsES(params martini.Params, r render.Render) {
	runid := params["runid"]
	fmt.Println(runid)
	url := os.Getenv("ES_URL") + "/logs/run/" + runid
	req, err := http.NewRequest("GET", url, nil)
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
	//                fmt.Println(string(bodybytes))

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(logs)
	var logtext string
	hrtimestamp := logs.Source.Hrtimestamp
	testid := logs.Source.Testid
	for _, line := range logs.Source.Logs {
		fmt.Println(hrtimestamp + " " + line)
		logtext = logtext + hrtimestamp + " " + " " + testid + " " + line + "\n"
	}

	r.Text(200, logtext)

}

func GetLogsESObj(params martini.Params, r render.Render) {
	runid := params["runid"]
	fmt.Println(runid)
	url := os.Getenv("ES_URL") + "/logs/run/" + runid
	req, err := http.NewRequest("GET", url, nil)
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
	//                fmt.Println(string(bodybytes))

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(logs)
	var logtext string
	hrtimestamp := logs.Source.Hrtimestamp
	var logobj []string
	for _, line := range logs.Source.Logs {
		fmt.Println(hrtimestamp + " " + line)
		logobj = append(logobj, logtext+hrtimestamp+" "+" "+line)
	}

	r.JSON(200, logobj)

}

func WriteLogESPost(eslogs structs.ESlogSpecIn1, berr binding.Errors, params martini.Params, r render.Render) {
	timestamp := int(int32(time.Now().Unix()))
	t := time.Now()
	hrtimestamp := t.Format(time.RFC3339)
	eslogs.Timestamp = timestamp
	eslogs.Hrtimestamp = hrtimestamp
	eslogs.Runid = params["runid"]
	p, err := json.Marshal(eslogs)
	if err != nil {
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("PUT", os.Getenv("ES_URL")+"/logs/run/"+params["runid"], bytes.NewBuffer(p))
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
	fmt.Println(string(bodybytes))
}

func GetRuns(params martini.Params, r render.Render) {

	runlist, err := getRuns(params["job"], params["jobspace"])
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
	}
	r.JSON(200, runlist)
}

func getRuns(job string, jobspace string) (rl structs.RunList, e error) {

	var runlist structs.RunList

	req, err := http.NewRequest("GET", os.Getenv("ES_URL")+"/logs/run/_search?q=job:"+job+"+AND+jobspace:"+jobspace+"&sort=timestamp:desc&size=50", nil)
	if err != nil {
		fmt.Println(err)
		return runlist, err
	}
	req.Header.Add("Content-type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return runlist, err
	}
	defer resp.Body.Close()
	bodybytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return runlist, err
	}
	fmt.Println(string(bodybytes))
	var runs structs.RunsSpec
	err = json.Unmarshal(bodybytes, &runs)
	if err != nil {
		fmt.Println(err)
		return runlist, err
	}
	for _, hit := range runs.Hits.Hits {
		if hit.Source.Job == job && hit.Source.Jobspace == jobspace {
			fmt.Println(hit.ID)
			fmt.Println(hit.Source.Job + "-" + hit.Source.Jobspace)
			fmt.Println(hit.Source.App + "-" + hit.Source.Space)
			fmt.Println(hit.Source.Hrtimestamp)
			fmt.Println(hit.Source.Overallstatus)
			fmt.Println("buildid:" + hit.Source.BuildID)
			var run structs.Run
			run.ID = hit.ID
			run.App = hit.Source.App
			run.Space = hit.Source.Space
			run.Job = hit.Source.Job
			run.Jobspace = hit.Source.Jobspace
			run.Hrtimestamp = hit.Source.Hrtimestamp
			run.Overallstatus = hit.Source.Overallstatus
			run.BuildID = hit.Source.BuildID
			runlist.Runs = append(runlist.Runs, run)
		}
	}
	fmt.Println(runlist)
	var cutlist structs.RunList
	//listlen := len(runlist.Runs)
	//startlen := listlen-10
	endlen := 11
	for index, element := range runlist.Runs {
		if index < endlen {
			cutlist.Runs = append(cutlist.Runs, element)
		}
	}
	cutlist.Runs = reverseRunList(cutlist.Runs)
	return cutlist, nil

}

func reverseRunList(input []structs.Run) []structs.Run {
	if len(input) == 0 {
		return input
	}
	return append(reverseRunList(input[1:]), input[0])
}

func GetRunInfo(params martini.Params, r render.Render) {
	runid := params["runid"]
	fmt.Println(runid)
	url := os.Getenv("ES_URL") + "/logs/run/" + runid
	req, err := http.NewRequest("GET", url, nil)
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
	//                fmt.Println(string(bodybytes))

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}
	var empty []string
	logs.Source.Logs = empty
	fmt.Println(logs)

	r.JSON(200, logs)

}
