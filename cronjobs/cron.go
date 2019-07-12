package cronjobs

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/robfig/cron"
	"net/http"
	dbstore "taas/dbstore"
	hooks "taas/hooks"
	structs "taas/structs"
)

var Cronjob *cron.Cron

func StartCron() {
	Cronjob = cron.New()
	Cronjob.Start()
	var cronjobs []structs.Cronjob
	cronjobs, err := dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range cronjobs {
		err := addCronjob(element.Job, element.Jobspace, element.FrequencyMinutes)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func GetCronjobs(params martini.Params, r render.Render) {
	var cronjobs []structs.Cronjob
	cronjobs, err := dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}
	r.JSON(200, cronjobs)
}
func AddCronjob(req *http.Request, params martini.Params, cronjob structs.Cronjob, berr binding.Errors, r render.Render) {
	//TODO: add audit

	if berr != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": berr})
	}
	err := dbstore.AddCronJob(cronjob)
	if err != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": err})
	}
	err = addCronjob(cronjob.Job, cronjob.Jobspace, cronjob.FrequencyMinutes)
	if err != nil {
		fmt.Println(berr)
		r.JSON(500, map[string]interface{}{"response": err})
	}
	r.JSON(200, map[string]interface{}{"status": "created"})
}

func addCronjob(job string, jobspace string, fm string) (e error) {
	diagnostic, err := dbstore.FindDiagnostic(job + "-" + jobspace)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var releasehookpayload structs.ReleaseHookSpec
	releasehookpayload.App.Name = diagnostic.App
	releasehookpayload.Space.Name = diagnostic.Space
	releasehookpayload.Action = "release"
	releasehookpayload.Release.Result = "succeeded"

	_, err = Cronjob.AddFunc("@every "+fm+"m", func() { hooks.ReleaseHookHandler(releasehookpayload, true) })
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("Added cronjob for: %v-%v\n", job, jobspace)
	return nil
}

func DeleteCronjob(req *http.Request, params martini.Params, r render.Render) {
	id := params["id"]
	deleteCronjob(id)
	r.JSON(200, map[string]interface{}{"status": "deleted"})

}

func deleteCronjob(id string) (e error) {
	cronjob, err := dbstore.GetCronjobByID(id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var entryid cron.EntryID
	for _, element := range Cronjob.Entries() {
		fmt.Printf("%+v\n", element)
		s := fmt.Sprintf("%v", element.Schedule)
		fmt.Println(s)
		if s == "{"+cronjob.FrequencyMinutes+"m0s}" {
			fmt.Println("found it")
			entryid = element.ID
			break
		}
	}
	Cronjob.Remove(entryid)
	err = dbstore.DeleteCronjob(id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
