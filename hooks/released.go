package hooks

import (
	"encoding/json"
	"fmt"
	"net/http"
	diagnostics "taas/diagnostics"
	githubapi "taas/githubapi"
	akkeris "taas/jobs"
	structs "taas/structs"
	"taas/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func ReleasedHook(req *http.Request, releasedhookpayload structs.ReleasedHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}
	ReleasedHookHandler(req, releasedhookpayload, false)
}

func ReleasedHookHandler(req *http.Request, releasedhookpayload structs.ReleasedHookSpec, isCron bool) {
	fmt.Println("*** ReleasedHookHandler ***")
	releasedhookpayload.Release.Result = "succeeded"
	s, _ := json.MarshalIndent(releasedhookpayload, "", "  ")
	fmt.Println(string(s))

	utils.PrintDebug(releasedhookpayload.App.Name)
	utils.PrintDebug(releasedhookpayload.Space.Name)
	utils.PrintDebug(releasedhookpayload.Action)
	utils.PrintDebug(releasedhookpayload.Release.Result)
	diagnosticslist, err := diagnostics.GetDiagnostics(releasedhookpayload.Space.Name, releasedhookpayload.App.Name, releasedhookpayload.Action, releasedhookpayload.Release.Result)
	if err != nil {
		fmt.Println(err)
	}
	for _, element := range diagnosticslist {
		element.ReleaseID = releasedhookpayload.Release.ID
		element.BuildID = releasedhookpayload.Build.ID
		element.Token = req.Header.Get("x-akkeris-token")
		version, err := akkeris.GetVersion(element.App, element.Space, element.BuildID)
		if err != nil {
			fmt.Println(err)
		}
		utils.PrintDebug(version)
		var commitauthor string
		var commitmessage string
		if version != "" {
			element.GithubVersion = version
			commitauthor, commitmessage, err = githubapi.GetCommitAuthor(version)
			if err != nil {
				fmt.Println(err)
			}
			utils.PrintDebug(commitauthor)
		} else {
			commitauthor = "none"
			commitmessage = "none"
		}
		org, err := akkeris.GetAppControllerOrg(element.App + "-" + element.Space)
		element.Organization = org
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		utils.PrintDebug(spew.Sdump(element))
		diagnostics.RunDiagnostic(element, isCron, structs.Cronjob{})
	}

}
