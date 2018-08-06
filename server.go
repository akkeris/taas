package main

import (
	diagnosticlogs "alamo-self-diagnostics/diagnosticlogs"
	diagnostics "alamo-self-diagnostics/diagnostics"
	hooks "alamo-self-diagnostics/hooks"
        dbstore "alamo-self-diagnostics/dbstore"
	structs "alamo-self-diagnostics/structs"
        jobs "alamo-self-diagnostics/alamo"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func main() {
        jobs.Startclient()
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/v1/releasehook", binding.Json(structs.ReleaseHookSpec{}), hooks.ReleaseHook)
        m.Post("/v1/buildhook", binding.Json(structs.BuildPayload{}), hooks.BuildHook)
	m.Post("/v1/diagnostic", binding.Json(structs.DiagnosticSpec{}), diagnostics.CreateDiagnostic)
        m.Patch("/v1/diagnostic", binding.Json(structs.DiagnosticSpec{}), diagnostics.UpdateDiagnostic)
	//	m.Get("/v1/diagnostic/jobspace/:jobspace/job/:job/logs", Logs)
	m.Get("/v1/diagnostic/:job/logs", diagnosticlogs.Logs)
	m.Get("/v1/diagnostic/jobspace/:jobspace/job/:job/logs", diagnosticlogs.LogsExtended)
        m.Get("/v1/diagnostic/jobspace/:jobspace/job/:job/runs", diagnosticlogs.GetRuns)
	m.Get("/v1/diagnostic/logs/:runid", diagnosticlogs.GetLogsES)
        m.Get("/v1/diagnostic/logs/:runid/array", diagnosticlogs.GetLogsESObj)
        m.Post("/v1/diagnostic/logs/:runid", binding.Json(structs.ESlogSpecIn1{}), diagnosticlogs.WriteLogESPost)       
	m.Post("/v1/diagnostic/:job/logs", binding.Json(structs.LogLines{}), diagnosticlogs.WriteLog)
	m.Post("/v1/diagnostic/jobspace/:jobspace/job/:job/logs", binding.Json(structs.LogLines{}), diagnosticlogs.WriteLogExtended)
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
	m.Use(martini.Static("static"))
	m.Run()
}
