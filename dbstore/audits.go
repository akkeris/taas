package dbstore

import (
	"encoding/json"
	"fmt"
	"os"
	structs "taas/structs"

	"database/sql"
	"net/http"
	auth "taas/auth"
	"time"

	"github.com/go-martini/martini"
	_ "github.com/lib/pq"
	"github.com/martini-contrib/render"
	uuid "github.com/nu7hatch/gouuid"
)

var db *sql.DB

func InitAuditPool() {
	uri := os.Getenv("DIAGNOSTICDB")
	var dberr error
	db, dberr = sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
		os.Exit(1)
	}

}
func AddConfigUnsetAudit(req *http.Request, id string, auditkey string) {
	audituuid, _ := uuid.NewV4()
	auditid := audituuid.String()
	user, err := auth.GetUser(req)
	if err != nil {
		fmt.Println(err)
	}
	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey) values ($1,$2,$3,$4,$5)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, id, user, "configvarunset", auditkey)
	if inserterr != nil {
		fmt.Println(inserterr)
	}
}

func AddConfigSetAudit(req *http.Request, id string, varspec structs.Varspec) {
	audituuid, _ := uuid.NewV4()
	auditid := audituuid.String()
	newvalue := varspec.Varvalue
	auditkey := varspec.Varname
	user, err := auth.GetUser(req)
	if err != nil {
		fmt.Println(err)
	}
	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey, newvalue) values ($1,$2,$3,$4,$5,$6)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, id, user, "configvarset", auditkey, newvalue)
	if inserterr != nil {
		fmt.Println(inserterr)
	}
}
func AddDiagnosticDeleteAudit(req *http.Request, diagnostic structs.DiagnosticSpec) {
	audituuid, _ := uuid.NewV4()
	auditid := audituuid.String()
	user, err := auth.GetUser(req)
	if err != nil {
		fmt.Println(err)
	}
	bodybytes, err := json.Marshal(diagnostic)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bodybytes))

	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey, newvalue) values ($1,$2,$3,$4,$5,$6)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, diagnostic.ID, user, "destroy", diagnostic.Job+"-"+diagnostic.JobSpace, string(bodybytes))
	if inserterr != nil {
		fmt.Println(inserterr)
	}
}

func AddDiagnosticCreateAudit(req *http.Request, diagnostic structs.DiagnosticSpec) {
	var diagnosticaudit structs.DiagnosticSpecAudit
	diagnosticaudit.ID = diagnostic.ID
	diagnosticaudit.App = diagnostic.App
	diagnosticaudit.Space = diagnostic.Space
	diagnosticaudit.Job = diagnostic.Job
	diagnosticaudit.JobSpace = diagnostic.JobSpace
	diagnosticaudit.Action = diagnostic.Action
	diagnosticaudit.Result = diagnostic.Result
	diagnosticaudit.Image = diagnostic.Image
	diagnosticaudit.PipelineName = diagnostic.PipelineName
	diagnosticaudit.TransitionFrom = diagnostic.TransitionFrom
	diagnosticaudit.TransitionTo = diagnostic.TransitionTo
	diagnosticaudit.Timeout = diagnostic.Timeout
	diagnosticaudit.Startdelay = diagnostic.Startdelay
	diagnosticaudit.Slackchannel = diagnostic.Slackchannel
	audituuid, _ := uuid.NewV4()
	auditid := audituuid.String()
	user, err := auth.GetUser(req)
	if err != nil {
		fmt.Println(err)
	}
	bodybytes, err := json.Marshal(diagnosticaudit)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bodybytes))

	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey, newvalue) values ($1,$2,$3,$4,$5, $6)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, diagnosticaudit.ID, user, "register", diagnostic.Job+"-"+diagnostic.JobSpace, string(bodybytes))
	if inserterr != nil {
		fmt.Println(inserterr)
	}

}

func AddDiagnosticUpdateAudit(req *http.Request, diagnostic structs.DiagnosticSpec) {
	var diagnosticaudit structs.DiagnosticSpecAudit
	diagnosticaudit.ID = diagnostic.ID
	diagnosticaudit.Image = diagnostic.Image
	diagnosticaudit.PipelineName = diagnostic.PipelineName
	diagnosticaudit.TransitionFrom = diagnostic.TransitionFrom
	diagnosticaudit.TransitionTo = diagnostic.TransitionTo
	diagnosticaudit.Timeout = diagnostic.Timeout
	diagnosticaudit.Startdelay = diagnostic.Startdelay
	diagnosticaudit.Slackchannel = diagnostic.Slackchannel
	audituuid, _ := uuid.NewV4()
	auditid := audituuid.String()
	user, err := auth.GetUser(req)
	if err != nil {
		fmt.Println(err)
	}
	bodybytes, err := json.Marshal(diagnosticaudit)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bodybytes))

	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey, newvalue) values ($1,$2,$3,$4,$5,$6)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, diagnosticaudit.ID, user, "properties", diagnostic.Job+"-"+diagnostic.JobSpace, string(bodybytes))
	if inserterr != nil {
		fmt.Println(inserterr)
	}

}

func GetAudits(params martini.Params, r render.Render) {
	provided := params["provided"]

	stmt, err := db.Prepare("select auditid, id, audituser, audittype, coalesce(auditkey,null,''), coalesce(newvalue,null,''), created_at from audits where id = $1 or auditkey = $1 order by created_at asc")
	if err != nil {
		fmt.Println(err)
	}
	defer stmt.Close()
	var audits []structs.Audit
	rows, err := stmt.Query(provided)
	for rows.Next() {
		var auditid string
		var id string
		var audituser string
		var audittype string
		var auditkey string
		var newvalue string
		var created_at time.Time
		err := rows.Scan(&auditid, &id, &audituser, &audittype, &auditkey, &newvalue, &created_at)
		if err != nil {
			fmt.Println(err)
		}
		var audit structs.Audit
		audit.Auditid = auditid
		audit.Id = id
		audit.Audituser = audituser
		audit.Audittype = audittype
		audit.Auditkey = auditkey
		audit.Newvalue = newvalue
		audit.Createdat = created_at
		audits = append(audits, audit)
	}
	r.JSON(200, audits)
}
