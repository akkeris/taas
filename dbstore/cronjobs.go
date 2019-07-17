package dbstore

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
	"os"
	structs "taas/structs"
        "time"
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

}

func StoreCronRun(diagnostic structs.DiagnosticSpec, starttime time.Time, endtime time.Time, cronid string) (e error) {

        var stmtstring string = "insert into cronruns (testid , runid , space , app , job , jobspace , image, overallstatus, starttime, endtime,cronid) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)"

        stmt, err := cdb.Prepare(stmtstring)
        if err != nil {
                db.Close()
                return err
        }

        _, inserterr := stmt.Exec(diagnostic.ID, diagnostic.RunID, diagnostic.Space, diagnostic.App, diagnostic.Job, diagnostic.JobSpace, diagnostic.Image, diagnostic.OverallStatus, starttime, endtime,cronid)

        if inserterr != nil {
                stmt.Close()
                return inserterr
        }
        stmt.Close()
        return nil
}

func GetCronjobRuns(id string, runs string) (j []structs.CronjobRun, e error) {
        var cronjobruns []structs.CronjobRun
        selectstring := "select * from (select starttime, endtime, overallstatus, runid from cronruns where cronid = $1 order by starttime desc limit "+runs+") as t order by starttime asc"
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
                current.Starttime= starttime
                current.Endtime=endtime
                current.Overallstatus=overallstatus
                current.RunID=runid
                cronjobruns = append(cronjobruns, current)
        }
        return cronjobruns, nil
}

func GetCronjobs() (j []structs.Cronjob, e error) {
	var cronjobs []structs.Cronjob
	selectstring := "select id, job, jobspace, cronspec from cronjobs"
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
		err := rows.Scan(&id, &job, &jobspace, &cs)
		if err != nil {
			fmt.Println(err)
		}
		current.ID = id
		current.Job = job
		current.Jobspace = jobspace
		current.Cronspec = cs
		cronjobs = append(cronjobs, current)
	}
	return cronjobs, nil
}

func AddCronJob(cronjob structs.Cronjob) (e error) {
	iduuid, _ := uuid.NewV4()
	id := iduuid.String()
	var stmtstring string = "insert into cronjobs (id, job,  jobspace, cronspec) values ($1,$2,$3,$4)"

	stmt, err := cdb.Prepare(stmtstring)
	if err != nil {
		fmt.Println(err)
		return err
	}

	_, inserterr := stmt.Exec(id, cronjob.Job, cronjob.Jobspace, cronjob.Cronspec)
	if inserterr != nil {
		fmt.Println(inserterr)
		return inserterr
	}
	return nil
}

func GetCronjobByID(id string) (c structs.Cronjob, e error) {
	var cronjob structs.Cronjob
	selectstring := "select id, job, jobspace, cronspec from cronjobs where id = $1"
	stmt, err := cdb.Prepare(selectstring)
	if err != nil {
		fmt.Println(err)
		return cronjob, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(id)
	for rows.Next() {
		var id string
		var job string
		var jobspace string
		var cs string
		err := rows.Scan(&id, &job, &jobspace, &cs)
		if err != nil {
			fmt.Println(err)
		}
		cronjob.ID = id
		cronjob.Job = job
		cronjob.Jobspace = jobspace
		cronjob.Cronspec = cs
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
