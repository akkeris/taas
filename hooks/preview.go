package hooks

import (
	"fmt"
	"os"
	"taas/dbstore"
	diagnostics "taas/diagnostics"
	githubapi "taas/githubapi"
	akkeris "taas/jobs"
	structs "taas/structs"

	uuid "github.com/nu7hatch/gouuid"

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

// PreviewCreatedHook - New preview app created. Register a diagnostic based on the new preview app
func PreviewCreatedHook(previewcreatedhookpayload structs.PreviewCreatedHookSpec, berr binding.Errors, r render.Render) {
	if berr != nil {
		fmt.Println("Bad Request")
		fmt.Println(berr)
		return
	}

	fmt.Println("Starting to create diagnostic...")

	diagnostic, err := dbstore.FindDiagnosticByApp(previewcreatedhookpayload.App.Name + "-" + previewcreatedhookpayload.Space.Name)
	if err != nil || diagnostic.ID == "" {
		fmt.Println(err)
		fmt.Println("Diagnostic not found for provided app. Ignoring preview-created hook.")
		return
	}

	d, err := dbstore.FindDiagnostic(previewcreatedhookpayload.Preview.App.Name + "-" + diagnostic.Job)
	if err == nil && d.ID != "" {
		fmt.Println("A diagnostic with the given name and space already exists. Ignoring preview-created hook.")
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}

	previewID, _ := uuid.NewV4()
	diagnostic.ID = previewID.String()
	// TODO: Job name can't be too long otherwise k8s throws an error with name over 63 characters
	diagnostic.Job = previewcreatedhookpayload.Preview.App.Name
	diagnostic.App = previewcreatedhookpayload.Preview.AppSetup.App.Name
	diagnostic.Space = previewcreatedhookpayload.Space.Name
	diagnostic.Action = "preview-released"
	diagnostic.PipelineName = "manual"
	diagnostic.TransitionFrom = "manual"
	diagnostic.TransitionTo = "manual"

	err = akkeris.CreateService(diagnostic)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Diagnostic " + diagnostic.Job + " created for " + previewcreatedhookpayload.Preview.AppSetup.App.Name + "-" + previewcreatedhookpayload.Space.Name)

	err = akkeris.CreateHook(true, []string{"destroy"}, os.Getenv("TAAS_SVC_URL")+"/v1/previewdestroyhook", "merpderp", previewcreatedhookpayload.Preview.App.Name)
	if err != nil {
		fmt.Println(err)
	}

	r.Text(200, "")
}

// PreviewDestroyHook - Delete the diagnostic associated with the preview app
func PreviewDestroyHook(payload structs.DestroyHookSpec, berr binding.Errors, r render.Render) {
	diagnostic, err := dbstore.FindDiagnosticByApp(payload.App.Name + "-" + payload.Space.Name)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = diagnostics.DeleteDiagnostic(diagnostic)
	if err != nil {
		fmt.Println(err)
		return
	}
}
