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
		panic(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT app, space, action, job, jobspace FROM diagnostics WHERE action=$1", "release")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var releaseDiagnostic structs.DiagnosticSpec
		err = rows.Scan(&releaseDiagnostic.App, &releaseDiagnostic.Space, &releaseDiagnostic.Action, &releaseDiagnostic.Job, &releaseDiagnostic.JobSpace)
		if err != nil {
			panic(err)
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
			panic(err)
		}

		for _, hook := range hooks {
			if strings.Contains(hook.URL, os.Getenv("TAAS_SVC_URL")+"/v1/releasehook") {
				fmt.Println("Deleting release webhook:\n  App: " + appspace + "\n  ID:  " + hook.ID + "\n  URL: " + hook.URL + "\n")
				err := jobs.DeleteHook(appspace, hook.ID)
				if err != nil {
					panic(err)
				}
			}
		}

		fmt.Println("Creating released webhook:\n  App: " + appspace + "\n")
		jobs.CreateHooks(appspace)

		releaseDiagnostic.Action = "released"
		jobs.UpdateService(releaseDiagnostic)
	}
}
