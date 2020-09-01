package cronjobs

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	dbstore "taas/dbstore"
	diagnostics "taas/diagnostics"
	structs "taas/structs"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
	"github.com/robfig/cron/v3"
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

func scheduleJob(diagnostic structs.DiagnosticSpec, cronjob structs.Cronjob) {
	diagnostics.RunDiagnostic(diagnostic, true, cronjob)
}

func GetCronjobRuns(req *http.Request, params martini.Params, r render.Render) {
	var cronjobruns []structs.CronjobRun
	var runs string
	var filter string
	runs = ""
	filter = ""
	if len(req.URL.Query()["runs"]) >= 1 {
		runs = req.URL.Query()["runs"][0]
	}
	if len(req.URL.Query()["filter"]) >= 1 {
		filter = req.URL.Query()["filter"][0]
	}
	if runs == "" {
		runs = "10"
	}
	if filter == "" {
		filter = "all"
	}
	cronjobruns, err := dbstore.GetCronjobRuns(params["id"], runs, filter)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	r.JSON(200, cronjobruns)

}

func GetCronjobs(params martini.Params, r render.Render) {
	if os.Getenv("ENABLE_CRON_WORKER") != "" {
		cronjobs, err := dbstore.GetCronjobsWithSchedule()
		if err != nil {
			fmt.Println(err)
			r.JSON(500, map[string]interface{}{"response": err})
			return
		}
		r.JSON(200, cronjobs)
		return
	}

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

	iduuid, _ := uuid.NewV4()
	cronjob.ID = iduuid.String()

	// Not using cron worker. Add to internal cron scheduler
	if os.Getenv("ENABLE_CRON_WORKER") == "" {
		err := addCronjob(req, cronjob)
		if err != nil {
			fmt.Println(berr)
			r.JSON(500, map[string]interface{}{"response": err.Error()})
			return
		}
	}

	err := dbstore.AddCronJob(cronjob)
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
	entryid, err := Cronjob.AddFunc(cronjob.Cronspec, func() { scheduleJob(diagnostic, cronjob) })
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
	err := deleteCronjob(req, id)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}
	r.JSON(200, map[string]interface{}{"status": "deleted"})
}

func deleteCronjob(req *http.Request, id string) (e error) {
	cronjob, err := dbstore.GetCronjobByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return errors.New("The specified cron job does not exist")
		}
		fmt.Println(err)
		return err
	}

	// Not using cron worker. Remove from internal scheduler
	if os.Getenv("ENABLE_CRON_WORKER") == "" {
		Cronjob.Remove(jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec])
		delete(jobmap, cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec)
	}

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

// UpdateCronjob updates the configuration for a cronjob
func UpdateCronjob(req *http.Request, params martini.Params, cronjob structs.Cronjob, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
		return
	}

	err := updateCronjob(req, params["id"], cronjob)
	if err != nil {
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}

	r.JSON(200, map[string]interface{}{"status": "updated"})
}

// updateCronjob currently only handles updates to disabled toggle (maintenance mode)
func updateCronjob(req *http.Request, id string, cronjob structs.Cronjob) (e error) {
	oldJob, err := dbstore.GetCronjobByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return errors.New("The specified cron job does not exist")
		}
		return err
	}

	if oldJob.Disabled != cronjob.Disabled {
		err := dbstore.UpdateCronjob(id, cronjob.Disabled)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCronjob gets a single cronjob from the database
func GetCronjob(req *http.Request, params martini.Params, r render.Render) {
	cronjob, err := dbstore.GetCronjobByID(params["id"])
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			r.JSON(500, map[string]interface{}{"response": "The specified cron job does not exist"})
			return
		}
		fmt.Println(err)
		r.JSON(500, map[string]interface{}{"response": err.Error()})
		return
	}

	r.JSON(200, cronjob)
}
