package hooks

import (
	"fmt"
	"taas/dbstore"
	diagnostics "taas/diagnostics"
	githubapi "taas/githubapi"
	akkeris "taas/jobs"
	structs "taas/structs"

	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func PreviewReleasedHook(previewreleasedhookpayload structs.PreviewReleasedHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}

	diagnosticslist, err := diagnostics.GetDiagnostics(previewreleasedhookpayload.Space.Name, previewreleasedhookpayload.App.Name, "preview-released", "succeeded")
	if err != nil {
		fmt.Println(err)
	}

	for _, element := range diagnosticslist {
		commitauthor, commitmessage, err := githubapi.GetCommitAuthor(previewreleasedhookpayload.Slug.SourceBlob.Commit)
		if err != nil {
			fmt.Println(err)
		}
		org, err := akkeris.GetAppControllerOrg(element.App + "-" + element.Space)
		if err != nil {
			fmt.Println(err)
		}
		element.Organization = org
		element.CommitAuthor = commitauthor
		element.CommitMessage = commitmessage
		diagnostics.RunDiagnostic(element)
	}
}

func PreviewCreatedHook(previewcreatedhookpayload structs.PreviewCreatedHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}

	diagnostic, err := dbstore.FindDiagnosticByApp(previewcreatedhookpayload.App.Name + "-" + previewcreatedhookpayload.Space.Name)
	if err != nil {
		fmt.Println(err)
	}

	diagnostic.App = previewcreatedhookpayload.Preview.App.Name
	diagnostic.Space = previewcreatedhookpayload.Space.Name
	diagnostic.Action = "preview-released"
	diagnostic.PipelineName = "manual"
	diagnostic.TransitionFrom = "manual"
	diagnostic.TransitionTo = "manual"

	err = akkeris.CreateService(diagnostic)
	if err != nil {
		fmt.Println(err)
	}

	r.Text(200, "")
}
