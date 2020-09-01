package dbstore

import (
	"database/sql"
	"fmt"
	"os"
	structs "taas/structs"
	"taas/utils"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
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
		err := rows.Scan(&current.Starttime, &current.Endtime, &current.Overallstatus, &current.RunID)
		if err != nil {
			fmt.Println(err)
			return cronjobruns, err
		}
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
	var stmtstring string = "insert into cronjobs (id, job, jobspace, cronspec, command, disabled) values ($1,$2,$3,$4,$5,$6)"

	stmt, err := cdb.Prepare(stmtstring)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, inserterr := stmt.Exec(cronjob.ID, cronjob.Job, cronjob.Jobspace, cronjob.Cronspec, cronjob.Command, cronjob.Disabled)
	if inserterr != nil {
		fmt.Println(inserterr)
		return inserterr
	}
	return nil
}

func GetCronjobByID(id string) (c structs.Cronjob, e error) {
	var cronjob structs.Cronjob
	var err error

	if os.Getenv("ENABLE_CRON_WORKER") != "" {
		query := `
			SELECT
			cronjobs.id AS id,
			cronjobs.job AS job,
			cronjobs.jobspace AS jobspace,
			cronjobs.cronspec AS cronspec,
			coalesce(cronjobs.command, null, '') AS command,
			cronjobs.disabled AS disabled,
			coalesce(cronjobschedule.next, null, '') AS next,
			coalesce(cronjobschedule.prev, null, '') AS prev
		FROM cronjobs
			LEFT JOIN cronjobschedule ON cronjobschedule.id = cronjobs.id
		WHERE cronjobs.id = $1
		`
		var next string
		var prev string

		err = cdb.QueryRow(query, id).Scan(&cronjob.ID, &cronjob.Job, &cronjob.Jobspace, &cronjob.Cronspec, &cronjob.Command, &cronjob.Disabled, &next, &prev)
		if err != nil {
			fmt.Println(err)
			return cronjob, err
		}

		if next != "" {
			cronjob.Next, err = time.Parse(time.RFC3339, next)
			if err != nil {
				fmt.Println("Could not parse next run time for " + cronjob.Job + "-" + cronjob.Jobspace)
			}
		}

		if prev != "" {
			cronjob.Prev, err = time.Parse(time.RFC3339, prev)
			if err != nil {
				fmt.Println("Could not parse last run time for " + cronjob.Job + "-" + cronjob.Jobspace)
			}
		}
		err = nil
	} else {
		err = cdb.QueryRow("select id, job, jobspace, cronspec from cronjobs where id = $1", id).Scan(&cronjob.ID, &cronjob.Job, &cronjob.Jobspace, &cronjob.Cronspec)
	}

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
	utils.PrintDebug(affect)

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

// UpdateCronjob currently only updates the disabled toggle
func UpdateCronjob(id string, disabled bool) (e error) {
	query := "update cronjobs set disabled = $2 where id = $1"
	_, err := cdb.Exec(query, id, disabled)
	if err != nil {
		return err
	}
	return nil
}

// CreateListener creates a new Postgres listener for the database
func CreateListener() *pq.Listener {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	return pq.NewListener(os.Getenv("DIAGNOSTICDB"), 10*time.Second, time.Minute, reportProblem)
}

// InsertCronScheduleEntry inserts an entry into the cron scheduler status table
func InsertCronScheduleEntry(id string, next time.Time, prev time.Time) error {
	query := "insert into cronjobschedule(id, next, prev) values($1, $2, $3)"

	nextBytes, _ := next.MarshalText()
	prevBytes, _ := prev.MarshalText()

	_, err := cdb.Exec(query, id, string(nextBytes), string(prevBytes))
	if err != nil {
		return err
	}
	return nil
}

// DeleteCronScheduleEntry deletes an entry from the cron scheduler status table
func DeleteCronScheduleEntry(id string) error {
	query := "delete from cronjobschedule where id = $1"
	_, err := cdb.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

// DeleteAllCronScheduleEntries wipes the cron scheduler status table
func DeleteAllCronScheduleEntries() error {
	_, err := cdb.Exec("delete from cronjobschedule")
	if err != nil {
		return err
	}
	return nil
}

// GetCronjobsWithSchedule gets cronjob information along with the current scheduled status
func GetCronjobsWithSchedule() ([]structs.Cronjob, error) {
	var entries []structs.Cronjob

	rows, err := db.Query(`
		SELECT
			cronjobs.id AS id,
			cronjobs.job AS job,
			cronjobs.jobspace AS jobspace,
			cronjobs.cronspec AS cronspec,
			coalesce(cronjobs.command, null, '') AS command,
			cronjobs.disabled AS disabled,
			coalesce(cronjobschedule.next, null, '') AS next,
			coalesce(cronjobschedule.prev, null, '') AS prev
		FROM cronjobs
			LEFT JOIN cronjobschedule ON cronjobschedule.id = cronjobs.id
	`)
	if err != nil {
		fmt.Println(err)
		return entries, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry structs.Cronjob
		var next string
		var prev string
		err := rows.Scan(&entry.ID, &entry.Job, &entry.Jobspace, &entry.Cronspec, &entry.Command, &entry.Disabled, &next, &prev)
		if err != nil {
			fmt.Println(err)
			return entries, err
		}

		if next != "" {
			entry.Next, err = time.Parse(time.RFC3339, next)
			if err != nil {
				fmt.Println("Could not parse next run time for " + entry.Job + "-" + entry.Jobspace)
			}
		}

		if prev != "" {
			entry.Prev, err = time.Parse(time.RFC3339, prev)
			if err != nil {
				fmt.Println("Could not parse last run time for " + entry.Job + "-" + entry.Jobspace)
			}
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

// UpdateCronScheduleEntry updates the cron schedule status for a given job
func UpdateCronScheduleEntry(id string, next time.Time, prev time.Time) error {
	query := "update cronjobschedule set next = $2, prev = $3 where id = $1"

	nextBytes, _ := next.MarshalText()
	prevBytes, _ := prev.MarshalText()

	_, err := cdb.Exec(query, id, string(nextBytes), string(prevBytes))
	if err != nil {
		return err
	}
	return nil
}
