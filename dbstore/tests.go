package dbstore

import (
	"database/sql"
	"fmt"
	"regexp"
	akkeris "taas/jobs"
	structs "taas/structs"

	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/go-martini/martini"
	_ "github.com/lib/pq"

	"github.com/martini-contrib/render"
)

func StoreRun(diagnostic structs.DiagnosticSpec) (e error) {

	fmt.Println("************************* dbstore")
	fmt.Println(diagnostic)
	fmt.Println("************************* dbstore")

	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}

	var stmtstring string = "insert into testruns (testid , runid , space , app , org , buildid , githubversion , commitauthor , commitmessage , action , result , job , jobspace , image , pipelinename , transitionfrom , transitionto , timeout, startdelay, overallstatus) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
		return err
	}

	_, inserterr := stmt.Exec(diagnostic.ID, diagnostic.RunID, diagnostic.Space, diagnostic.App, diagnostic.Organization, diagnostic.BuildID, diagnostic.GithubVersion, diagnostic.CommitAuthor, diagnostic.CommitMessage, diagnostic.Action, diagnostic.Result, diagnostic.Job, diagnostic.JobSpace, diagnostic.Image, diagnostic.PipelineName, diagnostic.TransitionFrom, diagnostic.TransitionTo, diagnostic.Timeout, diagnostic.Startdelay, diagnostic.OverallStatus)

	if inserterr != nil {
		stmt.Close()
		db.Close()
		return inserterr
	}
	stmt.Close()
	db.Close()
	return nil
}

func StoreBits(req *http.Request, params martini.Params, r render.Render) {
	runid := params["runid"]
	format := req.URL.Query().Get("format")
	fmt.Println(format)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	if format == "rspec" {
		err := storeBitsRspec(body, runid)
		if err != nil {
			r.JSON(500, map[string]interface{}{"response": err})
			return
		}
	}
	if format == "junit" {
		err := storeBitsJunit(body, runid)
		if err != nil {
			r.JSON(500, map[string]interface{}{"response": err})
			return
		}
	}
	if format == "" {
		err := storeBitsRspec(body, runid)
		if err != nil {
			r.JSON(500, map[string]interface{}{"response": err})
			return
		}
	}
	r.JSON(202, map[string]interface{}{"response": "accepted"})

}

func storeBitsRspec(requestbody []byte, runid string) (e error) {
	var bits structs.BitsRSpec
	err := json.Unmarshal(requestbody, &bits)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var id string

	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return dberr
	}
	defer db.Close()
	inserterr := db.QueryRow("insert into rspecsummary (runid, version, summaryline, duration, examplecount, failurecount, pendingcount, messages) values ($1,$2,$3,$4,$5,$6,$7,$8) returning runid;", runid, bits.Version, bits.SummaryLine, bits.Summary.Duration, bits.Summary.ExampleCount, bits.Summary.FailureCount, bits.Summary.PendingCount, strings.Join(bits.Messages, ",")).Scan(&id)

	if inserterr != nil {
		fmt.Println(inserterr)
		return inserterr
	}

	for _, element := range bits.Examples {
		inserterr = db.QueryRow("insert into rspecexamples (runid, id, description, fulldescription, status, filepath, linenumber, runtime, pendingmessage) values ($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id;", runid, element.ID, element.Description, element.FullDescription, element.Status, element.FilePath, element.LineNumber, element.RunTime, element.PendingMessage).Scan(&id)

		if inserterr != nil {
			fmt.Println(inserterr)
			return inserterr
		}

	}

	return nil

}

func storeBitsJunit(requestbody []byte, runid string) (e error) {

	var xmlstruct structs.Testsuite
	err := xml.Unmarshal(requestbody, &xmlstruct)
	if err != nil {
		fmt.Println(err)
		return err
	}
	//fmt.Println(xmlstruct.Testcases[0].Name)
	//fmt.Println(xmlstruct.Testcases[1].Name)
	var id string
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return dberr
	}
	defer db.Close()
	inserterr := db.QueryRow("insert into testsuite (runid, name, tests, failures, errors, time, timestamp, hostname) values ($1, $2, $3, $4, $5, $6, $7, $8) returning runid;", runid, xmlstruct.Name, xmlstruct.Tests, xmlstruct.Failures, xmlstruct.Errors, xmlstruct.Time, xmlstruct.Timestamp, xmlstruct.Hostname).Scan(&id)
	if inserterr != nil {
		fmt.Println(inserterr)
		return inserterr
	}
	for _, element := range xmlstruct.Testcases {

		inserterr = db.QueryRow("insert into testcase (runid, classname, name, file, time)  values ($1,$2,$3,$4,$5) returning runid;", runid, element.Classname, element.Name, element.File, element.Time).Scan(&id)

		if inserterr != nil {

			fmt.Println(inserterr)
			return inserterr
		}
	}
	return nil
}

func isUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

func FindDiagnosticByApp(app string) (d structs.DiagnosticSpec, e error) {
	var selectstring = "select id,  space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,''), coalesce(testpreviews,null,false) from diagnostics where app||'-'||space = $1"
	return findDiagnostic(app, selectstring)
}

func FindDiagnostic(provided string) (d structs.DiagnosticSpec, e error) {
	var selectstring string
	if !isUUID(provided) {
		selectstring = "select id,  space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,''), coalesce(testpreviews,null,false) from diagnostics where job||'-'||jobspace = $1"
	} else {
		selectstring = "select id,  space, app, action, result, job, jobspace, image, pipelinename, transitionfrom, transitionto, timeout, startdelay, slackchannel, coalesce(command,null,''), coalesce(testpreviews,null,false) from diagnostics where id = $1"
	}
	return findDiagnostic(provided, selectstring)
}

func findDiagnostic(input string, selectstring string) (d structs.DiagnosticSpec, e error) {
	var diagnostic structs.DiagnosticSpec
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return diagnostic, dberr
	}
	defer db.Close()

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
	rows, err := stmt.Query(input)
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
