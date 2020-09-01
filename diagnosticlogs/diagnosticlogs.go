package diagnosticlogs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	dbstore "taas/dbstore"
	structs "taas/structs"
	"taas/utils"
	"time"

	sarama "github.com/Shopify/sarama"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
)

type logObject struct {
	Log        string `json:"log"`
	Kubernetes struct {
		ContainerName string `json:"container_name"`
		NamespaceName string `json:"namespace_name"`
		PodName       string `json:"pod_name"`
	} `json:"kubernetes"`
	Topic string `json:"topic"`
}

type taasConsumerGroupHandler struct {
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Job      string
	JobSpace string
}

func (taasConsumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (taasConsumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h taasConsumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var logobj logObject
		err := json.Unmarshal(msg.Value, &logobj)
		if err != nil {
			fmt.Println(err)
		} else if logobj.Kubernetes.ContainerName == h.Job && logobj.Kubernetes.NamespaceName == h.JobSpace {
			event := logobj.Log
			fmt.Fprintf(h.Writer, event)
			h.Flusher.Flush()
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}

func GetLogs(space string, job string, instance string) (l []string, e error) {
	akkerisapiurl := os.Getenv("AKKERIS_API_URL")
	var lines []string
	req, err := http.NewRequest("GET", akkerisapiurl+"/v1/space/"+space+"/app/"+job+"/instance/"+instance+"/log", nil)
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
	lines = strings.Split(log.Logs, "\n")
	return lines, nil
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
	utils.PrintDebug(string(bodybytes))
}

func GetLogsES(params martini.Params, r render.Render) {
	runid := params["runid"]
	utils.PrintDebug(runid)
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

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}
	var logtext string
	zone, _ := time.LoadLocation(os.Getenv("LOG_TIMESTAMPS_LOCALE"))
	for _, line := range logs.Source.Logs {
		if line == "" {
			continue
		}
		mainpart := strings.Split(strings.Split(line, " ")[0], ".")[0]
		t, err := time.Parse("2006-01-02T15:04:05", mainpart)
		if err != nil {
			fmt.Println(err)
		}
		tinzone := t.In(zone)
		tinzonestring := fmt.Sprintf("%s", tinzone.Format("2006-01-02 03:04:05 PM"))
		logtext = logtext + "[" + tinzonestring + "]  " + strings.Join(strings.Split(line, " ")[1:], " ") + "\n"
	}

	r.Text(200, logtext)

}

func GetLogsESObj(params martini.Params, r render.Render) {
	runid := params["runid"]
	utils.PrintDebug(runid)
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

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}

	var logtext string
	zone, _ := time.LoadLocation(os.Getenv("LOG_TIMESTAMPS_LOCALE"))
	var logobj []string
	for _, line := range logs.Source.Logs {
		if line == "" {
			continue
		}
		mainpart := strings.Split(strings.Split(line, " ")[0], ".")[0]
		t, err := time.Parse("2006-01-02T15:04:05", mainpart)
		if err != nil {
			fmt.Println(err)
		}
		tinzone := t.In(zone)
		tinzonestring := fmt.Sprintf("%s", tinzone.Format("2006-01-02 03:04:05 PM"))
		logobj = append(logobj, logtext+"["+tinzonestring+"]  "+strings.Join(strings.Split(line, " ")[1:], " "))
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
	utils.PrintDebug(string(bodybytes))
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
	utils.PrintDebug(string(bodybytes))
	var runs structs.RunsSpec
	err = json.Unmarshal(bodybytes, &runs)
	if err != nil {
		fmt.Println(err)
		return runlist, err
	}
	for _, hit := range runs.Hits.Hits {
		if hit.Source.Job == job && hit.Source.Jobspace == jobspace {
			utils.PrintDebug(hit.ID)
			utils.PrintDebug(hit.Source.Job + "-" + hit.Source.Jobspace)
			utils.PrintDebug(hit.Source.App + "-" + hit.Source.Space)
			utils.PrintDebug(hit.Source.Hrtimestamp.String())
			utils.PrintDebug(hit.Source.Overallstatus)
			utils.PrintDebug("buildid:" + hit.Source.BuildID)
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
	utils.PrintDebug(runlist)
	var cutlist structs.RunList
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
	utils.PrintDebug(runid)
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

	var logs structs.ESlogSpecOut1
	err = json.Unmarshal(bodybytes, &logs)
	if err != nil {
		fmt.Println(err)
		return
	}
	var empty []string
	logs.Source.Logs = empty
	utils.PrintDebug(logs)

	r.JSON(200, logs)
}

func TailLogs(w http.ResponseWriter, req *http.Request, params martini.Params, r render.Render) {
	utils.PrintDebug(params["provided"])
	diagnostic, err := dbstore.FindDiagnostic(params["provided"])
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
	}
	if diagnostic.ID == "" {
		r.JSON(500, map[string]interface{}{"response": "invalid test"})
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	jobspace := diagnostic.JobSpace
	job := diagnostic.Job
	topic := jobspace
	cguuid, _ := uuid.NewV4()
	consumergroup := cguuid.String()
	brokers := os.Getenv("KAFKA_BROKERS")

	// New Sarama configuration
	config := sarama.NewConfig()
	version, err := sarama.ParseKafkaVersion("2.0.0")
	config.Version = version
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

	client, err := sarama.NewClient(strings.Split(brokers, ","), config)
	if err != nil {
		fmt.Println(err)
	}
	defer func() { _ = client.Close() }()

	// Start new consumer group
	group, err := sarama.NewConsumerGroupFromClient(consumergroup, client)
	if err != nil {
		log.Println(err)
	}
	defer func() { _ = group.Close() }()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Iterate over consumer sessions
	ctx := context.Background()
	for {
		topics := []string{topic}
		handler := &taasConsumerGroupHandler{}
		handler.Writer = w
		handler.Flusher = f
		handler.Job = job
		handler.JobSpace = jobspace
		err := group.Consume(ctx, topics, handler)
		if err != nil {
			log.Println(err)
		}
	}
}
