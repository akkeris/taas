package cronjobs

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
	"github.com/robfig/cron"
	"net/http"
	dbstore "taas/dbstore"
	diagnostics "taas/diagnostics"
	structs "taas/structs"
        uuid "github.com/nu7hatch/gouuid"
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
		err := addCronjob(element.Job, element.Jobspace, element.FrequencyMinutes)
		if err != nil {
			fmt.Println(err)
		}
	}
}

func scheduleJob(diagnostic structs.DiagnosticSpec){
    diagnostics.RunDiagnostic(diagnostic, true)
}

func GetCronjobs(params martini.Params, r render.Render) {
	var cronjobs []structs.Cronjob
        var newlist []structs.Cronjob
	cronjobs, err := dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}
        for _, element := range cronjobs {
            element.Next=Cronjob.Entry(jobmap[element.Job+"-"+element.Jobspace+"-"+element.FrequencyMinutes]).Next
            element.Prev=Cronjob.Entry(jobmap[element.Job+"-"+element.Jobspace+"-"+element.FrequencyMinutes]).Prev
            newlist = append(newlist, element)
        }
	r.JSON(200, newlist)
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
        runiduuid, _ := uuid.NewV4()
        runid := runiduuid.String()
        diagnostic.RunID=runid
        entryid, err := Cronjob.AddFunc("@every "+fm+"m", func() { scheduleJob(diagnostic) })
	if err != nil {
		fmt.Println(err)
		return err
	}
        jobmap[job+"-"+jobspace+"-"+fm]=entryid
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
	Cronjob.Remove(jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.FrequencyMinutes])
	err = dbstore.DeleteCronjob(id)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}
