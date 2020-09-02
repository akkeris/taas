package cronworker

import (
	"encoding/json"
	"fmt"
	"taas/dbstore"
	"taas/diagnostics"
	"taas/structs"
	"time"

	"github.com/lib/pq"
	"github.com/robfig/cron/v3"
	uuid "github.com/satori/go.uuid"
)

/*
	The cron worker is only responsible for scheduling and running cron diagnostics.
	It does not update configuration or add new cron jobs to the database.

	It listens to the diagnosticdb using the Postgres NOTIFY/LISTEN pattern.
	When records change in the cronjob table, it updates the internal cron scheduler.
*/

// Incoming message from the Postgres database
// Indicates a database record has changed
type incomingMessage struct {
	Table  string `json:"table"`
	Action string `json:"action"`
	Data   struct {
		OldRecord structs.Cronjob `json:"old_record"`
		NewRecord structs.Cronjob `json:"new_record"`
	} `json:"data"`
}

var cronScheduler *cron.Cron
var jobmap map[string]cron.EntryID
var listener *pq.Listener

// Start starts the cron scheduler and listens for changed database records
func Start() {
	fmt.Println("[cron_worker]: Initializing...")

	// Initialize the cron scheduler
	cronScheduler = cron.New()
	cronScheduler.Start()

	dbstore.InitCronjobPool()

	jobmap = make(map[string]cron.EntryID)

	// Remove any old scheduled runs in database
	err := dbstore.DeleteAllCronScheduleEntries()
	if err != nil {
		fmt.Println(err)
	}

	// Get a list of current database entries and add to cron scheduler
	fmt.Println("[cron_worker]: Adding existing cronjobs from the database (if any)...")
	var cronjobs []structs.Cronjob
	cronjobs, err = dbstore.GetCronjobs()
	if err != nil {
		fmt.Println(err)
	}

	for _, element := range cronjobs {
		if element.Disabled {
			continue
		}
		err := addCronjob(element)
		if err != nil {
			fmt.Println(err)
		}
	}

	// Create database listener
	listener = dbstore.CreateListener()
	err = listener.Listen("events")
	if err != nil {
		panic(err)
	}

	fmt.Println("[cron_worker]: Initialization complete!")
	fmt.Println("[cron_worker]: Watching for database changes...")

	// Wait and handle new messages from the database
	for {
		waitForChanges()
	}
}

// Handle incoming database messages
func waitForChanges() {
	for {
		select {

		// Message received
		case n := <-listener.Notify:
			// n will be nil when the connection was reestablished
			if n == nil {
				fmt.Println("[cron_worker]: Connection reestablished")
				return
			}

			// Unmarshal to struct
			var msg incomingMessage
			err := json.Unmarshal([]byte(n.Extra), &msg)
			if err != nil {
				fmt.Println("Error unmarshalling JSON: ", err)
				return
			}

			fmt.Println()
			if msg.Action == "DELETE" {
				deleteCronjob(msg.Data.OldRecord)
			} else if msg.Action == "INSERT" {
				addCronjob(msg.Data.NewRecord)
			} else if msg.Action == "UPDATE" {
				updateCronjob(msg.Data.OldRecord, msg.Data.NewRecord)
			}

			return

		// It's been a while since we talked to the database, check and make sure we're still connected
		case <-time.After(90 * time.Second):
			go func() {
				err := listener.Ping()
				if err != nil {
					fmt.Println("Error checking database connection: " + err.Error())
				}
			}()
			return
		}
	}
}

// Add a new cron job to the scheduler
func addCronjob(cronjob structs.Cronjob) (e error) {
	// Should not add disabled jobs to the scheduler
	if cronjob.Disabled {
		return nil
	}
	fmt.Println("[cron_worker]: Scheduling cronjob for " + cronjob.Job + "-" + cronjob.Jobspace + " (" + cronjob.ID + ")...")

	diagnostic, err := dbstore.FindDiagnostic(cronjob.Job + "-" + cronjob.Jobspace)
	if err != nil {
		fmt.Println("\n" + err.Error())
		return err
	}

	runid := uuid.NewV4()
	diagnostic.RunID = runid.String()

	entryid, err := cronScheduler.AddFunc(cronjob.Cronspec, func() {
		entryid := jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec]
		diagnostics.RunDiagnostic(diagnostic, true, cronjob)
		err := dbstore.UpdateCronScheduleEntry(cronjob.ID, cronScheduler.Entry(entryid).Next, cronScheduler.Entry(entryid).Prev)
		if err != nil {
			fmt.Println("\nError updating scheduler information in the database")
			fmt.Println(err)
		}
	})

	if err != nil {
		fmt.Println("\n" + err.Error())
		return err
	}

	jobmap[cronjob.Job+"-"+cronjob.Jobspace+"-"+cronjob.Cronspec] = entryid

	err = dbstore.InsertCronScheduleEntry(cronjob.ID, cronScheduler.Entry(entryid).Next, cronScheduler.Entry(entryid).Prev)
	if err != nil {
		fmt.Println("\nError adding scheduler information to the database")
		fmt.Println(err)
	}

	fmt.Println("[cron_worker]: Cronjob added! Next run in " + time.Until(cronScheduler.Entry(entryid).Next).String())
	return nil
}

// Remove a cron job from the scheduler
func deleteCronjob(oldRecord structs.Cronjob) {
	// Disabled jobs won't be in the scheduler
	if oldRecord.Disabled {
		return
	}
	fmt.Println("[cron_worker]: Removing cronjob " + oldRecord.Job + "-" + oldRecord.Jobspace + " (" + oldRecord.ID + ")")
	cronScheduler.Remove(jobmap[oldRecord.Job+"-"+oldRecord.Jobspace+"-"+oldRecord.Cronspec])
	delete(jobmap, oldRecord.Job+"-"+oldRecord.Jobspace+"-"+oldRecord.Cronspec)
	err := dbstore.DeleteCronScheduleEntry(oldRecord.ID)
	if err != nil {
		fmt.Println("Error removing scheduler information from the database")
		fmt.Println(err)
	}
	fmt.Println("[cron_worker]: Cronjob removed.")
}

// Update a cron job in the scheduler
func updateCronjob(oldRecord structs.Cronjob, newRecord structs.Cronjob) {
	fmt.Println("[cron_worker]: Cronjob " + oldRecord.ID + " has been updated. Removing old job from the schedule...")
	deleteCronjob(oldRecord)
	if newRecord.Disabled {
		fmt.Println("[cron_worker]: Cronjob " + oldRecord.ID + " was disabled, not rescheduling.")
	} else {
		addCronjob(newRecord)
	}
	fmt.Println("[cron_worker]: Update complete.")
}
