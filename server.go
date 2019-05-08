package main

import (
	"fmt"
	"os"
	"strconv"
	dbstore "taas/dbstore"
	diagnosticlogs "taas/diagnosticlogs"
	diagnostics "taas/diagnostics"
	hooks "taas/hooks"
	jobs "taas/jobs"
	structs "taas/structs"

	artifacts "taas/artifacts"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func checkEnv() {
	var c int
	var requiredEnv = []string{
		"AKKERIS_API_URL", "APP_CONTROLLER_AUTH_SECRET", "APP_CONTROLLER_URL",
		"DEFAULT_ORG", "DEFAULT_START_DELAY", "DIAGNOSTICDB", "ENABLE_SLACK_NOTIFICATIONS",
		"ES_URL", "GITHUB_TOKEN_SECRET", "KIBANA_URL", "KUBERNETES_API_SERVER",
		"KUBERNETES_TOKEN_SECRET", "LOG_URL", "PITDB", "POSTBACKURL", "RERUN_URL",
		"SLACK_NOTIFICATION_CHANNEL_DEFAULT", "SLACK_NOTIFICATION_URL",
		"VAULT_ADDR", "VAULT_TOKEN",
	}

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

func main() {
	checkEnv()
	artifacts.Init()
	jobs.Startclient()
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/v1/releasehook", binding.Json(structs.ReleaseHookSpec{}), hooks.ReleaseHook)
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
	m.Delete("/v1/diagnostic/:provided", diagnostics.DeleteDiagnostic)
	m.Post("/v1/diagnostic/:provided/bind/**", diagnostics.BindDiagnosticSecret)
	m.Delete("/v1/diagnostic/:provided/bind/**", diagnostics.UnbindDiagnosticSecret)

	m.Post("/v1/diagnostic/:provided/config", binding.Json(structs.Varspec{}), diagnostics.SetConfig)
	m.Delete("/v1/diagnostic/:provided/config/:varname", diagnostics.UnsetConfig)

	m.Post("/v1/diagnostic/results/:runid", dbstore.StoreBits)
	m.Get("/octhc", diagnostics.Octhc)

	m.Get("/v1/artifacts/:runid/", artifacts.Wrapper(artifacts.Awss3))
	m.Get("/v1/artifacts/:runid/**", artifacts.Wrapper(artifacts.Awss3))

	m.Use(martini.Static("static"))
	m.Run()
}
