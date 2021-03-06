package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	artifacts "taas/artifacts"
	cronjobs "taas/cronjobs"
	"taas/cronworker"
	dbstore "taas/dbstore"
	diagnosticlogs "taas/diagnosticlogs"
	diagnostics "taas/diagnostics"
	hooks "taas/hooks"
	jobs "taas/jobs"
	"taas/migrations"
	structs "taas/structs"
	"taas/utils"

	"github.com/go-martini/martini"
	"github.com/joho/godotenv"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func checkEnv() {
	var c int
	var requiredEnv = []string{"AKKERIS_API_URL", "APP_CONTROLLER_AUTH", "APP_CONTROLLER_URL", "DEFAULT_ORG", "DEFAULT_START_DELAY", "DIAGNOSTICDB", "ENABLE_SLACK_NOTIFICATIONS", "ES_URL", "GITHUB_TOKEN", "KIBANA_URL", "KUBERNETES_API_SERVER", "KUBERNETES_API_VERSION", "KUBERNETES_CLIENT_TYPE", "KUBERNETES_TOKEN", "LOG_URL", "PITDB", "PORT", "POSTBACKURL", "RERUN_URL", "SLACK_NOTIFICATION_CHANNEL", "SLACK_NOTIFICATION_URL", "VAULT_ADDR", "VAULT_TOKEN", "AWS_S3_BUCKET", "AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "DIRECTORY_LISTINGS", "STRIP_PATH", "DIRECTORY_LISTINGS_FORMAT", "ARTIFACTS_URL", "LOG_TIMESTAMPS_LOCALE", "TAAS_SVC_URL", "AUTH_HOST"}

	for _, env := range requiredEnv {
		_, set := os.LookupEnv(env)
		if !set {
			fmt.Println("Environment variable " + env + " missing!")
			c++
		}
	}

	if c > 0 {
		fmt.Println(strconv.Itoa(c) + " required environment variables missing. Exiting...")
		os.Exit(1)
	}
}

func createDB() {
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println("Error: Unable to run migration scripts - could not establish connection to DIAGNOSTICDB")
		fmt.Println(dberr)
		os.Exit(-1)
	}
	defer db.Close()

	buf, err := ioutil.ReadFile("./create.sql")
	if err != nil {
		fmt.Println("Error: Unable to run migration scripts - could not load create.sql")
		fmt.Println(err)
		os.Exit(-1)
	}

	_, err = db.Exec(string(buf))
	if err != nil {
		fmt.Println("Error: Unable to run migration scripts - execution failed")
		fmt.Println(err)
		os.Exit(-1)
	}
}

func isCronWorker() bool {
	for _, v := range os.Args[1:] {
		if v == "--cron_worker" {
			return true
		}
	}
	return false
}

func main() {
	godotenv.Load()
	checkEnv()
	createDB()
	dbstore.InitAuditPool()
	dbstore.InitCronjobPool()
	artifacts.Init()
	jobs.StartClient()

	if isCronWorker() {
		cronworker.Start()
		return
	}

	// Not using new cron worker
	if os.Getenv("ENABLE_CRON_WORKER") == "" {
		cronjobs.StartCron()
	}

	// Mark orphan jobs (env should be set if running in production)
	if os.Getenv("FIND_ORPHANS") != "" {
		dbstore.FindOrphans()
		dbstore.FindCronOrphans()
	}

	if os.Getenv("MIGRATE_TO_RELEASED") != "" {
		migrations.MigrateToReleased()
	}

	m := utils.CreateClassicMartini()
	m.Use(render.Renderer())
	m.Post("/v1/releasehook", binding.Json(structs.ReleaseHookSpec{}), hooks.ReleaseHook)
	m.Post("/v1/releasedhook", binding.Json(structs.ReleasedHookSpec{}), hooks.ReleasedHook)
	m.Post("/v1/buildhook", binding.Json(structs.BuildPayload{}), hooks.BuildHook)
	m.Post("/v1/diagnostic", binding.Json(structs.DiagnosticSpec{}), diagnostics.CreateDiagnostic)
	m.Patch("/v1/diagnostic", binding.Json(structs.DiagnosticSpec{}), diagnostics.UpdateDiagnostic)
	m.Get("/v1/diagnostic/jobspace/:jobspace/job/:job/runs", diagnosticlogs.GetRuns)
	m.Get("/v1/diagnostic/logs/:runid", diagnosticlogs.GetLogsES)
	m.Get("/v1/diagnostic/logs/:runid/array", diagnosticlogs.GetLogsESObj)
	m.Post("/v1/diagnostic/logs/:runid", binding.Json(structs.ESlogSpecIn1{}), diagnosticlogs.WriteLogESPost)
	m.Get("/v1/diagnostics", diagnostics.GetDiagnosticsList)
	m.Get("/v1/diagnostics/runs/info/:runid", diagnosticlogs.GetRunInfo)
	m.Get("/v1/diagnostic/rerun", diagnostics.Rerun)
	m.Get("/v1/diagnostic/:provided", diagnostics.GetDiagnosticByNameOrID)
	m.Delete("/v1/diagnostic/:provided", diagnostics.HTTPDeleteDiagnostic)
	m.Post("/v1/diagnostic/:provided/bind/**", diagnostics.BindDiagnosticSecret)
	m.Delete("/v1/diagnostic/:provided/bind/**", diagnostics.UnbindDiagnosticSecret)

	m.Post("/v1/diagnostic/:provided/config", binding.Json(structs.Varspec{}), diagnostics.SetConfig)
	m.Delete("/v1/diagnostic/:provided/config/:varname", diagnostics.UnsetConfig)

	m.Post("/v1/diagnostic/results/:runid", dbstore.StoreBits)
	m.Get("/octhc", diagnostics.Octhc)

	m.Get("/v1/artifacts/:runid/", artifacts.Wrapper(artifacts.Awss3))
	m.Get("/v1/artifacts/:runid/**", artifacts.Wrapper(artifacts.Awss3))

	m.Post("/v1/diagnostic/:provided/hooks", diagnostics.CreateHooks)

	m.Get("/v1/diagnostic/:provided/audits", dbstore.GetAudits)

	m.Get("/v1/diagnostic/:provided/taillogs", diagnosticlogs.TailLogs)

	m.Post("/v1/previewreleasedhook", binding.Json(structs.PreviewReleasedHookSpec{}), hooks.PreviewReleasedHook)
	m.Post("/v1/previewcreatedhook", binding.Json(structs.PreviewCreatedHookSpec{}), hooks.PreviewCreatedHook)
	m.Post("/v1/previewdestroyhook", binding.Json(structs.DestroyHookSpec{}), hooks.PreviewDestroyHook)

	m.Get("/v1/cronjobs", cronjobs.GetCronjobs)
	m.Post("/v1/cronjob", binding.Json(structs.Cronjob{}), cronjobs.AddCronjob)
	m.Delete("/v1/cronjob/:id", cronjobs.DeleteCronjob)
	m.Get("/v1/cronjob/:id/runs", cronjobs.GetCronjobRuns)

	m.Get("/v1/status/runs", diagnostics.GetCurrentRuns)

	if os.Getenv("ENABLE_CRON_WORKER") != "" {
		m.Get("/v1/cronjob/:id", cronjobs.GetCronjob)
		m.Patch("/v1/cronjob/:id", binding.Json(structs.Cronjob{}), cronjobs.UpdateCronjob)
	}

	m.Use(martini.Static("static"))
	m.Run()
}
