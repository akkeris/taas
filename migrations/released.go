package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"taas/jobs"
	"taas/structs"
)

// MigrateToReleased migrates all diagnostics that are triggered on the `release` webhook to the `released` webhook.
func MigrateToReleased() {
	var releaseDiagnostics []structs.DiagnosticSpec
	db, err := sql.Open("postgres", os.Getenv("DIAGNOSTICDB"))
	if err != nil {
		fmt.Println("Error opening DB")
		panic(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT app, space, action, job, jobspace FROM diagnostics WHERE action=$1", "release")
	if err != nil {
		fmt.Println("Error selecting all diagnostics with action=release")
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var releaseDiagnostic structs.DiagnosticSpec
		err = rows.Scan(&releaseDiagnostic.App, &releaseDiagnostic.Space, &releaseDiagnostic.Action, &releaseDiagnostic.Job, &releaseDiagnostic.JobSpace)
		if err != nil {
			fmt.Println("Error scanning rows")
			fmt.Println(err)
			continue
		}
		releaseDiagnostics = append(releaseDiagnostics, releaseDiagnostic)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}

	for _, releaseDiagnostic := range releaseDiagnostics {
		appspace := releaseDiagnostic.App + "-" + releaseDiagnostic.Space
		hooks, err := jobs.GetHooks(appspace)
		if err != nil {
			fmt.Println("Error: Unable to get hook. Diagnostic " + releaseDiagnostic.Job + "-" + releaseDiagnostic.JobSpace + ". App: " + appspace)
			fmt.Println(err)
		}

		for _, hook := range hooks {
			if strings.Contains(hook.URL, os.Getenv("TAAS_SVC_URL")+"/v1/releasehook") {
				fmt.Println("Deleting release webhook:\n  App: " + appspace + "\n  ID:  " + hook.ID + "\n  URL: " + hook.URL + "\n")
				err := jobs.DeleteHook(appspace, hook.ID)
				if err != nil {
					fmt.Println("Error: Unable to delete hook. Diagnostic " + releaseDiagnostic.Job + "-" + releaseDiagnostic.JobSpace + ". App: " + appspace + " Hook ID: " + hook.ID + ". Hook URL: " + hook.URL)
					fmt.Println(err)
				}
			}
		}

		fmt.Println("Creating released webhook:\n  App: " + appspace + "\n")
		jobs.CreateHooks(appspace)

		sqlStatement := `
			UPDATE diagnostics
			SET action='released'
			WHERE job=$1 and jobspace=$2`
		_, err = db.Exec(sqlStatement, releaseDiagnostic.Job, releaseDiagnostic.JobSpace)
		if err != nil {
			fmt.Println("Error: Unable to update diagnostic. Diagnostic " + releaseDiagnostic.Job + "-" + releaseDiagnostic.JobSpace + ". App: " + appspace)
			fmt.Println(err)
			continue
		}
	}
}
