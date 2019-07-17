package cronjobs

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/robfig/cron"
	"net/http"
	dbstore "taas/dbstore"
	diagnostics "taas/diagnostics"
	structs "taas/structs"
)

var Cronjob *cron.Cron
var jobmap map[string]cron.EntryID

func StartCron() {
	jobmap = make(map[string]cron.EntryID)

	Cronjob = cron.New()
	Cronjob.Start()
	var cronjobs []structs.Cronjob
	cronjobs, err := dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range cronjobs {
		err := addCronjob(nil, element)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func scheduleJob(diagnostic structs.DiagnosticSpec, cronid string) {
	diagnostics.RunDiagnostic(diagnostic, true, cronid)
}

func GetCronjobRuns(req *http.Request, params martini.Params, r render.Render) {
	var cronjobruns []structs.CronjobRun
	var runs string
	if len(req.URL.Query()["runs"]) < 1 {
		runs = "10"
	} else {
		runs = req.URL.Query()["runs"][0]
	}
	cronjobruns, err := dbstore.GetCronjobRuns(params["id"], runs)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	r.JSON(200, cronjobruns)

}

func GetCronjobs(params martini.Params, r render.Render) {
	var cronjobs []structs.Cronjob
	var newlist []structs.Cronjob
	cronjobs, err := dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range cronjobs {
		element.Next = Cronjob.Entry(jobmap[element.Job+"-"+element.Jobspace+"-"+element.Cronspec]).Next
		element.Prev = Cronjob.Entry(jobmap[element.Job+"-"+element.Jobspace+"-"+element.Cronspec]).Prev
		newlist = append(newlist, element)
	}
	r.JSON(200, newlist)
}
func AddCronjob(req *http.Request, params martini.Params, cronjob structs.Cronjob, berr binding.Errors, r render.Render) {
	//TODO: add audit

	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
		return
	}
	fmt.Printf("%+v\n", cronjob)
	err := addCronjob(req, cronjob)
	if err != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	err = dbstore.AddCronJob(cronjob)
	if err != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": err})
		return
	}
	r.JSON(200, map[string]interface{}{"status": "created"})
}

func addCronjob(req *http.Request, cronjob structs.Cronjob) (e error) {
	diagnostic, err := dbstore.FindDiagnostic(cronjob.Job + "-" + cronjob.Jobspace)
	if err != nil {
		fmt.Println(err)
		return err
	}
	runiduuid, _ := uuid.NewV4()
	runid := runiduuid.String()
	diagnostic.RunID = runid
	entryid, err := Cronjob.AddFunc(cronjob.Cronspec, func() { scheduleJob(diagnostic, cronjob.ID) })
	if err != nil {
		fmt.Println(err)
		return err
	}
	jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec] = entryid
	fmt.Printf("Added cronjob for: %v-%v\n", cronjob.Job, cronjob.Jobspace)
	if req != nil {
		dbstore.AddCronjobCreateAudit(req, diagnostic.ID, cronjob)
	}
	return nil
}

func DeleteCronjob(req *http.Request, params martini.Params, r render.Render) {
	id := params["id"]
	deleteCronjob(req, id)
	r.JSON(200, map[string]interface{}{"status": "deleted"})

}

func deleteCronjob(req *http.Request, id string) (e error) {
	cronjob, err := dbstore.GetCronjobByID(id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	Cronjob.Remove(jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec])
	err = dbstore.DeleteCronjob(id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	diagnostic, err := dbstore.FindDiagnostic(cronjob.Job + "-" + cronjob.Jobspace)
	if err != nil {
		fmt.Println(err)
		return err
	}
	dbstore.AddCronjobDeleteAudit(req, diagnostic.ID, cronjob)
	return nil
}
