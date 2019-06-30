package dbstore

import (
	"encoding/json"
	"fmt"
	"os"
	structs "taas/structs"

	"database/sql"
	_ "github.com/lib/pq"
	uuid "github.com/nu7hatch/gouuid"
	"net/http"
	auth "taas/auth"
)

func AddConfigUnsetAudit(req *http.Request, id string, auditkey string){
        audituuid, _ := uuid.NewV4()
        auditid := audituuid.String()
        user, err := auth.GetUser(req)
        if err != nil {
                fmt.Println(err)
        }
        uri := os.Getenv("DIAGNOSTICDB")
        db, dberr := sql.Open("postgres", uri)
        if dberr != nil {
                fmt.Println(dberr)
        }
        var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey) values ($1,$2,$3,$4,$5)"

        stmt, err := db.Prepare(stmtstring)
        if err != nil {
                db.Close()
        }

        _, inserterr := stmt.Exec(auditid, id, user, "configvarunset",auditkey)
        if inserterr != nil {
                fmt.Println(inserterr)
        }
 }

func AddConfigSetAudit(req *http.Request, id string, varspec structs.Varspec){
        audituuid, _ := uuid.NewV4()
        auditid := audituuid.String()
        newvalue := varspec.Varvalue
        auditkey := varspec.Varname
        user, err := auth.GetUser(req)
        if err != nil {
                fmt.Println(err)
        }
        uri := os.Getenv("DIAGNOSTICDB")
        db, dberr := sql.Open("postgres", uri)
        if dberr != nil {
                fmt.Println(dberr)
        }
        var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, auditkey, newvalue) values ($1,$2,$3,$4,$5,$6)"

        stmt, err := db.Prepare(stmtstring)
        if err != nil {
                db.Close()
        }

        _, inserterr := stmt.Exec(auditid, id, user, "configvarset",auditkey,newvalue)
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
        uri := os.Getenv("DIAGNOSTICDB")
        db, dberr := sql.Open("postgres", uri)
        if dberr != nil {
                fmt.Println(dberr)
        }

        var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, newvalue) values ($1,$2,$3,$4,$5)"

        stmt, err := db.Prepare(stmtstring)
        if err != nil {
                db.Close()
        }

        _, inserterr := stmt.Exec(auditid, diagnosticaudit.ID, user, "register", string(bodybytes))
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
	uri := os.Getenv("DIAGNOSTICDB")
	db, dberr := sql.Open("postgres", uri)
	if dberr != nil {
		fmt.Println(dberr)
	}

	var stmtstring string = "insert into audits (auditid,  id, audituser, audittype, newvalue) values ($1,$2,$3,$4,$5)"

	stmt, err := db.Prepare(stmtstring)
	if err != nil {
		db.Close()
	}

	_, inserterr := stmt.Exec(auditid, diagnosticaudit.ID, user, "properties", string(bodybytes))
	if inserterr != nil {
		fmt.Println(inserterr)
	}

}
