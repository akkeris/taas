package dbstore

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
	"os"
	structs "taas/structs"
)

func GetCronjobs() (j []structs.Cronjob, e error) {
	var cronjobs []structs.Cronjob
	selectstring := "select id, job, jobspace, frequency_minutes from cronjobs"
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return cronjobs, dberr
	}
	stmt, err := db.Prepare(selectstring)
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
		var fm string
		err := rows.Scan(&id, &job, &jobspace, &fm)
		if err != nil {
			fmt.Println(err)
		}
		current.ID = id
		current.Job = job
		current.Jobspace = jobspace
		current.FrequencyMinutes = fm
		cronjobs = append(cronjobs, current)
	}
	return cronjobs, nil
}

func AddCronJob(cronjob structs.Cronjob) (e error) {
	iduuid, _ := uuid.NewV4()
	id := iduuid.String()
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return dberr
	}
	var stmtstring string = "insert into cronjobs (id, job,  jobspace, frequency_minutes) values ($1,$2,$3,$4)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
		fmt.Println(err)
		return err
	}

	_, inserterr := stmt.Exec(id, cronjob.Job, cronjob.Jobspace, cronjob.FrequencyMinutes)
	if inserterr != nil {
		db.Close()
		fmt.Println(inserterr)
		return inserterr
	}
	return nil
}

func GetCronjobByID(id string) (c structs.Cronjob, e error) {
	var cronjob structs.Cronjob
	selectstring := "select id, job, jobspace, frequency_minutes from cronjobs where id = $1"
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		return cronjob, dberr
	}
	stmt, err := db.Prepare(selectstring)
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
		var fm string
		err := rows.Scan(&id, &job, &jobspace, &fm)
		if err != nil {
			fmt.Println(err)
		}
		cronjob.ID = id
		cronjob.Job = job
		cronjob.Jobspace = jobspace
		cronjob.FrequencyMinutes = fm
	}
	return cronjob, nil
}

func DeleteCronjob(id string) (e error) {
	uri := os.Getenv("DIAGNOSTICDB")
	db, err := sql.Open("postgres", uri)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer db.Close()
	stmt, err := db.Prepare("delete from cronjobs where id=$1")
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

	db.Close()

	return nil

}
