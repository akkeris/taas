package dbstore

import (
	"database/sql"
	"fmt"
	"os"
	structs "taas/structs"
	"time"

	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
)

var cdb *sql.DB

func InitCronjobPool() {
	uri := os.Getenv("DIAGNOSTICDB")
	var dberr error
	cdb, dberr = sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		os.Exit(1)
	}
	cdb.SetMaxOpenConns(15)
}

// FindCronOrphans - If jobs are in the "starting" or "running" state from any previous TaaS instance, mark them as orphans
func FindCronOrphans() {
	_, err := cdb.Exec("update cronruns set overallstatus='orphaned' where overallstatus='starting' or overallstatus='running'")
	if err != nil {
		fmt.Println(err)
	}
}

func StoreCronRun(diagnostic structs.DiagnosticSpec, starttime time.Time, endtime *time.Time, cronid string) (e error) {
	stmtstring := "insert into cronruns (testid , runid , space , app , job , jobspace , image, overallstatus, starttime, endtime,cronid) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)"

	stmt, err := cdb.Prepare(stmtstring)
	if err != nil {
		db.Close()
		return err
	}

	var inserterr error
	if endtime == nil {
		_, inserterr = stmt.Exec(diagnostic.ID, diagnostic.RunID, diagnostic.Space, diagnostic.App, diagnostic.Job, diagnostic.JobSpace, diagnostic.Image, diagnostic.OverallStatus, starttime, nil, cronid)
	} else {
		_, inserterr = stmt.Exec(diagnostic.ID, diagnostic.RunID, diagnostic.Space, diagnostic.App, diagnostic.Job, diagnostic.JobSpace, diagnostic.Image, diagnostic.OverallStatus, starttime, *endtime, cronid)
	}

	if inserterr != nil {
		stmt.Close()
		return inserterr
	}
	stmt.Close()
	return nil
}

func UpdateCronRun(diagnostic structs.DiagnosticSpec, endtime *time.Time) (e error) {
	var err error
	if endtime == nil {
		_, err = cdb.Exec("update cronruns set overallstatus=$1 where runid=$2", diagnostic.OverallStatus, diagnostic.RunID)
	} else {
		_, err = cdb.Exec("update cronruns set overallstatus=$1, endtime=$2 where runid=$3", diagnostic.OverallStatus, *endtime, diagnostic.RunID)
	}
	if err != nil {
		return err
	}
	return nil
}

func GetCronjobRuns(id string, runs string, filter string) (j []structs.CronjobRun, e error) {
	var cronjobruns []structs.CronjobRun
	var selectstring string
	if filter == "all" {
		selectstring = "select * from (select starttime, endtime, overallstatus, runid from cronruns where cronid = $1 order by starttime desc limit " + runs + ") as t order by starttime asc"
	}
	if filter != "all" {
		selectstring = "select * from (select starttime, endtime, overallstatus, runid from cronruns where cronid = $1 and overallstatus = '" + filter + "' order by starttime desc limit " + runs + ") as t order by starttime asc"
	}
	stmt, err := cdb.Prepare(selectstring)
	if err != nil {
		fmt.Println(err)
		return cronjobruns, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(id)
	for rows.Next() {
		var current structs.CronjobRun
		var starttime time.Time
		var endtime time.Time
		var overallstatus string
		var runid string
		err := rows.Scan(&starttime, &endtime, &overallstatus, &runid)
		if err != nil {
			return cronjobruns, err
			fmt.Println(err)
		}
		current.Starttime = starttime
		current.Endtime = endtime
		current.Overallstatus = overallstatus
		current.RunID = runid
		cronjobruns = append(cronjobruns, current)
	}
	return cronjobruns, nil
}

func GetCronjobs() (j []structs.Cronjob, e error) {
	var cronjobs []structs.Cronjob
	selectstring := "select id, job, jobspace, cronspec, coalesce(command,null,'') from cronjobs"
	stmt, err := cdb.Prepare(selectstring)
	if err != nil {
		fmt.Println(err)
		return cronjobs, err
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	for rows.Next() {
		var current structs.Cronjob
		var id string
		var job string
		var jobspace string
		var cs string
		var command string
		err := rows.Scan(&id, &job, &jobspace, &cs, &command)
		if err != nil {
			fmt.Println(err)
		}
		current.ID = id
		current.Job = job
		current.Jobspace = jobspace
		current.Cronspec = cs
		current.Command = command
		cronjobs = append(cronjobs, current)
	}
	return cronjobs, nil
}

func AddCronJob(cronjob structs.Cronjob) (e error) {
	iduuid, _ := uuid.NewV4()
	id := iduuid.String()
	var stmtstring string = "insert into cronjobs (id, job,  jobspace, cronspec, command) values ($1,$2,$3,$4,$5)"

	stmt, err := cdb.Prepare(stmtstring)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, inserterr := stmt.Exec(id, cronjob.Job, cronjob.Jobspace, cronjob.Cronspec, cronjob.Command)
	if inserterr != nil {
		fmt.Println(inserterr)
		return inserterr
	}
	return nil
}

func GetCronjobByID(id string) (c structs.Cronjob, e error) {
	var cronjob structs.Cronjob

	err := cdb.QueryRow("select id, job, jobspace, cronspec from cronjobs where id = $1", id).Scan(&cronjob.ID, &cronjob.Job, &cronjob.Jobspace, &cronjob.Cronspec)
	if err != nil {
		fmt.Println(err)
		return cronjob, err
	}

	return cronjob, nil
}

func DeleteCronjob(id string) (e error) {
	stmt, err := cdb.Prepare("delete from cronjobs where id=$1")
	if err != nil {
		fmt.Println(err)
		return err
	}
	res, err := stmt.Exec(id)
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

	return nil
}

func GetCurrentCronRuns() (r []structs.PendingCronRun, e error) {
	var runs []structs.PendingCronRun

	rows, err := cdb.Query("select runid, testid, cronid, app, space, job, jobspace, image, overallstatus, starttime, endtime from cronruns where overallstatus='starting' or overallstatus='running'")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var run structs.PendingCronRun
		err := rows.Scan(&run.RunID, &run.TestID, &run.CronID, &run.App, &run.Space, &run.Job, &run.Jobspace, &run.Image, &run.Overallstatus, &run.StartTime, &run.EndTime)
		if err != nil {
			fmt.Println(err)
			return runs, err
		}
		runs = append(runs, run)
	}

	return runs, nil
}
