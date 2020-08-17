package structs

import (
	"time"

	"github.com/lib/pq"
)

type Releases []struct {
	App struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"app"`
	CreatedAt   time.Time `json:"created_at"`
	Description string    `json:"description"`
	Slug        struct {
		ID string `json:"id"`
	} `json:"slug"`
	ID     string `json:"id"`
	Status string `json:"status"`
	State  string `json:"state"`
	User   struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	Version int  `json:"version"`
	Current bool `json:"current"`
}

type Statuses struct {
	State   string `json:"state"`
	Release struct {
		App struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"app"`
		CreatedAt   time.Time `json:"created_at"`
		Description string    `json:"description"`
		Slug        struct {
			ID string `json:"id"`
		} `json:"slug"`
		ID     string `json:"id"`
		Status string `json:"status"`
		State  string `json:"state"`
		User   struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
		Version int  `json:"version"`
		Current bool `json:"current"`
	} `json:"release"`
	Statuses []struct {
		ID          string    `json:"id"`
		State       string    `json:"state"`
		Name        string    `json:"name"`
		Context     string    `json:"context"`
		Description string    `json:"description"`
		TargetURL   string    `json:"target_url"`
		ImageURL    string    `json:"image_url"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"statuses"`
}

type ReleaseStatus struct {
	State       string `json:"state"`
	Name        string `json:"name"`
	Context     string `json:"context"`
	TargetURL   string `json:"target_url"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
}

type LogLines struct {
	Logs []string `json:"logs"`
}

type Target struct {
	App struct {
		ID string `json:"id"`
	} `json:"app"`
}

type PromotionSpec struct {
	Pipeline struct {
		ID string `json:"id"`
	} `json:"pipeline"`
	Source struct {
		App struct {
			ID string `json:"id"`
		} `json:"app"`
	} `json:"source"`
	Targets []Target `json:"targets"`
}

type PipelineSpec []struct {
	App struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"app"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Stage     string    `json:"stage"`
	Pipeline  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pipeline"`
}

type ActionSpec struct {
	Messages []string `json:"messages,omitempty"`
	Name     string   `json:"name"`
	Status   string   `json:"status"`
}

type StepSpec struct {
	Name         string       `json:"name"`
	Organization string       `json:org"`
	Actions      []ActionSpec `json:"actions"`
}

type ResultSpec struct {
	Payload struct {
		StartTime       string     `json:"start_time"`
		StopTime        string     `json:"stop_time"`
		BuildTimeMillis int64      `json:"build_time_millis"`
		Lifecycle       string     `json:"lifecycle"`
		Outcome         string     `json:"outcome"`
		Status          string     `json:"status"`
		Steps           []StepSpec `json:"steps"`
	} `json:"payload"`
}

type InstanceStatusSpec []struct {
	Instanceid string    `json:"instanceid"`
	Phase      string    `json:"phase"`
	Starttime  time.Time `json:"starttime"`
	Reason     string    `json:"reason"`
	Appstatus  []struct {
		App         string    `json:"app"`
		Readystatus bool      `json:"readystatus"`
		Startedat   time.Time `json:"startedat"`
	} `json:"appstatus"`
}

type JobRunSpec struct {
	Image                 string `json:"image"`
	DeleteBeforeCreate    bool   `json:"deleteBeforeCreate"`
	RestartPolicy         string `json:"restartPolicy"`
	ActiveDeadlineSeconds int    `json:"activeDeadlineSeconds"`
}

type ReleaseHookSpec struct {
	Action string `json:"action"`
	App    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
	Release struct {
		ID          string    `json:"id"`
		Result      string    `json:"result"`
		CreatedAt   time.Time `json:"created_at"`
		Version     int       `json:"version"`
		Description string    `json:"description"`
	} `json:"release"`
	Build struct {
		ID string `json:"id"`
	} `json:"build"`
}

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DiagnosticSpec struct {
	ID             string                `json:"id"`
	Space          string                `json:"space"`
	App            string                `json:"app"`
	Organization   string                `json:"org"`
	BuildID        string                `json:"buildid"`
	ReleaseID      string                `json:"releaseid"`
	GithubVersion  string                `json:"version"`
	CommitAuthor   string                `json:"commitauthor"`
	CommitMessage  string                `json:"commitmessage"`
	Action         string                `json:"action"`
	Result         string                `json:"result"`
	Job            string                `json:"job"`
	JobSpace       string                `json:"jobspace"`
	Image          string                `json:"image"`
	PipelineName   string                `json:"pipelinename"`
	TransitionFrom string                `json:"transitionfrom"`
	TransitionTo   string                `json:"transitionto"`
	Timeout        int                   `json:"timeout"`
	Startdelay     int                   `json:"startdelay"`
	Slackchannel   string                `json:"slackchannel"`
	Env            []EnvironmentVariable `json:"env"`
	RunID          string                `json:"runid"`
	OverallStatus  string                `json:"overallstatus"`
	Command        string                `json:"command"`
	TestPreviews   bool                  `json:"testpreviews"` // Run diagnostic on the app's preview apps
	IsPreview      bool                  `json:"ispreview"`    // This diagnostic is for a preview app
	Token          string                `json:"token"`
	WebhookURLs    string                `json:"webhookurls"` // Comma-separated list of webhook URLs to notify with test results
}

type DiagnosticSpecAudit struct {
	ID             string `json:"id"`
	Space          string `json:"space,omitempty"`
	App            string `json:"app,omitempty"`
	Action         string `json:"action,omitempty"`
	Result         string `json:"result,omitempty"`
	Job            string `json:"job,omitempty"`
	JobSpace       string `json:"jobspace,omitempty"`
	Image          string `json:"image"`
	PipelineName   string `json:"pipelinename"`
	TransitionFrom string `json:"transitionfrom"`
	TransitionTo   string `json:"transitionto"`
	Timeout        int    `json:"timeout"`
	Startdelay     int    `json:"startdelay"`
	Slackchannel   string `json:"slackchannel"`
	Command        string `json:"command"`
}

type ESlogSpecIn struct {
	Job           string   `json:"job"`
	Jobspace      string   `json:"jobspace"`
	App           string   `json:"app"`
	Space         string   `json:"space"`
	Testid        string   `json:"testid"`
	Timestamp     int      `json:"timestamp"`
	Hrtimestamp   string   `json:"hrtimestamp"`
	Logs          []string `json:"logs"`
	RunID         string   `json:"runid"`
	OverallStatus string   `json:"overallstatus"`
	BuildID       string   `json:"buildid"`
	Organization  string   `json:"org"`
}

type ESlogSpecOut struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total    int     `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string  `json:"_index"`
			Type   string  `json:"_type"`
			ID     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source struct {
				Job         string   `json:"job"`
				Jobspace    string   `json:"jobspace"`
				App         string   `json:"app"`
				Space       string   `json:"space"`
				Testid      string   `json:"testid"`
				Timestamp   string   `json:"timestamp"`
				Hrtimestamp string   `json:"hrtimestamp"`
				Logs        []string `json:"logs"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type ESlogSpecOut1 struct {
	Index   string `json:"_index"`
	Type    string `json:"_type"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Found   bool   `json:"found"`
	Source  struct {
		Job         string   `json:"job"`
		Jobspace    string   `json:"jobspace"`
		App         string   `json:"app"`
		Space       string   `json:"space"`
		Testid      string   `json:"testid"`
		Timestamp   int      `json:"timestamp"`
		Hrtimestamp string   `json:"hrtimestamp"`
		BuildID     string   `json:"buildid"`
		Logs        []string `json:"logs"`
	} `json:"_source"`
}

type Varspec struct {
	Setname  string `json:"setname"`
	Varname  string `json:"varname"`
	Varvalue string `json:"varvalue"`
}

type ESlogSpecIn1 struct {
	Job           string   `json:"job"`
	Jobspace      string   `json:"jobspace"`
	App           string   `json:"app"`
	Space         string   `json:"space"`
	Testid        string   `json:"testid"`
	Timestamp     int      `json:"timestamp"`
	Hrtimestamp   string   `json:"hrtimestamp"`
	Logs          []string `json:"logs"`
	Runid         string   `json:"runid"`
	Overallstatus string   `json:"overallstatus"`
}

type BuildSpec struct {
	App struct {
		ID string `json:"id"`
	} `json:"app"`
	Buildpacks      interface{} `json:"buildpacks"`
	CreatedAt       time.Time   `json:"created_at"`
	ID              string      `json:"id"`
	OutputStreamURL string      `json:"output_stream_url"`
	SourceBlob      struct {
		Checksum string `json:"checksum"`
		URL      string `json:"url"`
		Version  string `json:"version"`
		Commit   string `json:"commit"`
	} `json:"source_blob"`
	Release interface{} `json:"release"`
	Slug    struct {
		ID string `json:"id"`
	} `json:"slug"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

type CommitSpec struct {
	Sha    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
		Tree    struct {
			Sha string `json:"sha"`
			URL string `json:"url"`
		} `json:"tree"`
		URL          string `json:"url"`
		CommentCount int    `json:"comment_count"`
	} `json:"commit"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	CommentsURL string `json:"comments_url"`
	Author      struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"author"`
	Committer struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		AvatarURL         string `json:"avatar_url"`
		GravatarID        string `json:"gravatar_id"`
		URL               string `json:"url"`
		HTMLURL           string `json:"html_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		OrganizationsURL  string `json:"organizations_url"`
		ReposURL          string `json:"repos_url"`
		EventsURL         string `json:"events_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		Type              string `json:"type"`
		SiteAdmin         bool   `json:"site_admin"`
	} `json:"committer"`
	Parents []struct {
		Sha     string `json:"sha"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	} `json:"parents"`
	Stats struct {
		Total     int `json:"total"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
	Files []struct {
		Sha         string `json:"sha"`
		Filename    string `json:"filename"`
		Status      string `json:"status"`
		Additions   int    `json:"additions"`
		Deletions   int    `json:"deletions"`
		Changes     int    `json:"changes"`
		BlobURL     string `json:"blob_url"`
		RawURL      string `json:"raw_url"`
		ContentsURL string `json:"contents_url"`
		Patch       string `json:"patch"`
	} `json:"files"`
}

type ConfigSpec []struct {
	Setname  string `json:"setname"`
	Varname  string `json:"varname"`
	Varvalue string `json:"varvalue"`
}

type Run struct {
	ID            string    `json:"id"`
	App           string    `json:"app"`
	Space         string    `json:"space"`
	Job           string    `json:"job"`
	Jobspace      string    `json:"jobspace"`
	Hrtimestamp   time.Time `json:"hrtimestamp"`
	Overallstatus string    `json:"overallstatus"`
	BuildID       string    `json:"buildid"`
}

type RunList struct {
	Runs []Run `json:"runs"`
}

type RunsSpec struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total    int     `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string  `json:"_index"`
			Type   string  `json:"_type"`
			ID     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source struct {
				Job           string    `json:"job"`
				Jobspace      string    `json:"jobspace"`
				App           string    `json:"app"`
				Space         string    `json:"space"`
				Testid        string    `json:"testid"`
				Timestamp     int       `json:"timestamp"`
				Hrtimestamp   time.Time `json:"hrtimestamp"`
				Logs          []string  `json:"logs"`
				Runid         string    `json:"runid"`
				Overallstatus string    `json:"overallstatus"`
				BuildID       string    `json:"buildid"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type AppControllerApp struct {
	ArchivedAt                   time.Time `json:"archived_at"`
	BuildpackProvidedDescription string    `json:"buildpack_provided_description"`
	BuildStack                   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"build_stack"`
	CreatedAt   time.Time `json:"created_at"`
	GitURL      string    `json:"git_url"`
	ID          string    `json:"id"`
	Maintenance bool      `json:"maintenance"`
	Name        string    `json:"name"`
	SimpleName  string    `json:"simple_name"`
	Key         string    `json:"key"`
	Owner       struct {
		Email string `json:"email"`
		ID    string `json:"id"`
	} `json:"owner"`
	Organization struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"organization"`
	Formation struct {
	} `json:"formation"`
	Region struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"region"`
	ReleasedAt time.Time `json:"released_at"`
	RepoSize   int       `json:"repo_size"`
	SlugSize   int       `json:"slug_size"`
	Space      struct {
		Name string `json:"name"`
	} `json:"space"`
	Stack struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"stack"`
	UpdatedAt time.Time `json:"updated_at"`
	WebURL    string    `json:"web_url"`
}

type BuildPayload struct {
	Action string `json:"action"`
	App    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
	Build struct {
		ID        string    `json:"id"`
		Result    string    `json:"result"`
		CreatedAt time.Time `json:"created_at"`
		Repo      string    `json:"repo"`
		Commit    string    `json:"commit"`
	} `json:"build"`
}

type BuildInfo struct {
	App struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
	CreatedAt  time.Time `json:"created_at"`
	ID         string    `json:"id"`
	SourceBlob struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	} `json:"source_blob"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"updated_at"`
	Organization string    `json:"org"`
}

type BuildESSend struct {
	App          string    `json:"app"`
	Space        string    `json:"space"`
	Organization string    `json:"org"`
	ID           string    `json:"buildid"`
	Version      string    `json:"version"`
	Commit       string    `json:"commit"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"hrtimestamp"`
}

type Bindspec struct {
	App      string `json:"appname"`
	Space    string `json:"space"`
	Bindtype string `json:"bindtype"`
	Bindname string `json:"bindname"`
}

type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BitsRSpec struct {
	Version  string   `json:"version"`
	Messages []string `json:"messages"`
	Examples []struct {
		ID              string      `json:"id"`
		Description     string      `json:"description"`
		FullDescription string      `json:"full_description"`
		Status          string      `json:"status"`
		FilePath        string      `json:"file_path"`
		LineNumber      int         `json:"line_number"`
		RunTime         float64     `json:"run_time"`
		PendingMessage  interface{} `json:"pending_message"`
	} `json:"examples"`
	Summary struct {
		Duration     float64 `json:"duration"`
		ExampleCount int     `json:"example_count"`
		FailureCount int     `json:"failure_count"`
		PendingCount int     `json:"pending_count"`
	} `json:"summary"`
	SummaryLine string `json:"summary_line"`
}

type Testsuite struct {
	Skipped   string     `xml:"skipped,attr"`
	Timestamp string     `xml:"timestamp,attr"`
	Tests     string     `xml:"tests,attr"`
	Failures  string     `xml:"failures,attr"`
	Time      string     `xml:"time,attr"`
	Name      string     `xml:"name,attr"`
	Errors    string     `xml:"errors,attr"`
	Hostname  string     `xml:"hostname,attr"`
	Testcases []Testcase `xml:"testcase"`
	Property  Property   `xml:"properties>property"`
}

type Testcase struct {
	File      string `xml:"file,attr"`
	Time      string `xml:"time,attr"`
	Classname string `xml:"classname,attr"`
	Name      string `xml:"name,attr"`
}
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type AppInfo struct {
	Appname   string `json:"appname"`
	Space     string `json:"space"`
	Instances int    `json:"instances"`
	Bindings  []struct {
		Appname  string `json:"appname"`
		Space    string `json:"space"`
		Bindtype string `json:"bindtype"`
		Bindname string `json:"bindname"`
	} `json:"bindings"`
	Plan        string `json:"plan"`
	Healthcheck string `json:"healthcheck"`
	Image       string `json:"image"`
}

type SpaceInfo struct {
	Compliance []string  `json:"compliance"`
	CreatedAt  time.Time `json:"created_at"`
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Region     struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"region"`
	Stack struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"stack"`
	State     string    `json:"state"`
	Apps      string    `json:"apps"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OneOffSpec struct {
	Space         string                `json:"space"`
	Podname       string                `json:"podname"`
	Containername string                `json:"containername"`
	Image         string                `json:"image"`
	Command       string                `json:"command,omitempty"`
	Env           []EnvironmentVariable `json:"env"`
}

type OneOffPod struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name   string `json:"name"`
		Labels struct {
			Name  string `json:"name"`
			Space string `json:"space"`
		} `json:"labels"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		Containers                    []ContainerItem `json:"containers"`
		ImagePullPolicy               string          `json:"imagePullPolicy,omitempty"`
		ImagePullSecrets              []SecretItem    `json:"imagePullSecrets"`
		RestartPolicy                 string          `json:"restartPolicy"`
		TerminationGracePeriodSeconds int             `json:"terminationGracePeriodSeconds"`
	} `json:"spec"`
}

type ContainerItem struct {
	Name             string                `json:"name"`
	Image            string                `json:"image"`
	Command          []string              `json:"command,omitempty"`
	Args             []string              `json:"args,omitempty"`
	Env              []EnvironmentVariable `json:"env,omitempty"`
	ImagePullPolicy  string                `json:"imagePullPolicy,omitempty"`
	ImagePullSecrets []SecretItem          `json:"imagePullSecrets,omitempty"`
}

type SecretItem struct {
	Name string `json:"name"`
}

type AppHook struct {
	ID        string    `json:"id,omitempty"`
	Active    bool      `json:"active"`
	Events    []string  `json:"events"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
}
type PromoteStatus struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Pipeline  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pipeline"`
	Source struct {
		App struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"app"`
		Space struct {
			Name string `json:"name"`
		} `json:"space"`
		Release struct {
			ID string `json:"id"`
		} `json:"release"`
	} `json:"source"`
	ID     string `json:"id"`
	Status string `json:"status"`
}

type AuthUser struct {
	Email string `json:"email"`
	Cn    string `json:"cn"`
}

type Audit struct {
	Auditid   string    `json:"auditid"`
	Id        string    `json:"id"`
	Audituser string    `json:"audituser"`
	Audittype string    `json:"audittype"`
	Auditkey  string    `json:"auditkey"`
	Newvalue  string    `json:"newvalue"`
	Createdat time.Time `json:"created_at"`
}

type PodDescribe struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	Spec struct {
		Containers []struct {
			Name  string `json:"name"`
			Image string `json:"image"`
		} `json:"containers"`
		NodeName string `json:"nodeName"`
	} `json:"spec"`
	Status struct {
		Phase      string `json:"phase"`
		Conditions []struct {
			Type    string `json:"type"`
			Status  string `json:"status"`
			Reason  string `json:"reason,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"conditions"`
		StartTime time.Time `json:"startTime"`
	} `json:"status"`
	Events EventList `json:"events"`
}

type EventList struct {
	Items []struct {
		LastTimestamp time.Time `json:"lastTimestamp"`
		Reason        string    `json:"reason"`
		Message       string    `json:"message"`
		Type          string    `json:"type"`
	} `json:"items"`
}

type TemplatePod struct {
	Name       string
	Space      string
	Node       string
	StartTime  time.Time
	Status     string
	Containers []struct {
		Name  string `json:"name"`
		Image string `json:"image"`
	}
	Conditions []struct {
		Type    string `json:"type"`
		Status  string `json:"status"`
		Reason  string `json:"reason,omitempty"`
		Message string `json:"message,omitempty"`
	}
	Events []struct {
		LastTimestamp time.Time `json:"lastTimestamp"`
		Reason        string    `json:"reason"`
		Message       string    `json:"message"`
		Type          string    `json:"type"`
	}
}

type PreviewReleasedHookSpec struct {
	App struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
	Dyno struct {
		Type string `json:"type"`
	} `json:"dyno"`
	Key    string `json:"key"`
	Action string `json:"action"`
	Slug   struct {
		Image      string `json:"image"`
		SourceBlob struct {
			Checksum string `json:"checksum"`
			URL      string `json:"url"`
			Version  string `json:"version"`
			Commit   string `json:"commit"`
			Author   string `json:"author"`
			Repo     string `json:"repo"`
			Branch   string `json:"branch"`
			Message  string `json:"message"`
		} `json:"source_blob"`
		ID string `json:"id"`
	} `json:"slug"`
	ReleasedAt time.Time `json:"released_at"`
	Release    struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Version   int       `json:"version"`
	} `json:"release"`
	SourceApp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"source_app"`
}

type PreviewCreatedHookSpec struct {
	Action string `json:"action"`
	App    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
	Change  string `json:"change"`
	Preview struct {
		App struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"app"`
		AppSetup struct {
			ID        string    `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			App       struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"app"`
			Build struct {
				ID              interface{} `json:"id"`
				Status          string      `json:"status"`
				OutputStreamURL interface{} `json:"output_stream_url"`
			} `json:"build"`
			Progress           int           `json:"progress"`
			Status             string        `json:"status"`
			FailureMessage     string        `json:"failure_message"`
			ManifestErrors     []interface{} `json:"manifest_errors"`
			Postdeploy         interface{}   `json:"postdeploy"`
			ResolvedSuccessURL interface{}   `json:"resolved_success_url"`
		} `json:"app_setup"`
		Sites []struct {
			ID     string `json:"id"`
			Domain string `json:"domain"`
			Region struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"region"`
			CreatedAt  time.Time     `json:"created_at"`
			UpdatedAt  time.Time     `json:"updated_at"`
			Compliance []interface{} `json:"compliance"`
		} `json:"sites"`
	} `json:"preview"`
}

type DestroyHookSpec struct {
	Action string `json:"action"`
	App    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"app"`
	Space struct {
		Name string `json:"name"`
	} `json:"space"`
}

type Cronjob struct {
	ID       string    `json:"id"`
	Job      string    `json:"job"`
	Jobspace string    `json:"jobspace"`
	Cronspec string    `json:"cs"`
	Command  string    `json:"command"`
	Prev     time.Time `json:"prev"`
	Next     time.Time `json:"next"`
}

type CronjobRun struct {
	Starttime     time.Time   `json:"starttime"`
	Endtime       pq.NullTime `json:"endtime"`
	Overallstatus string      `json:"overallstatus"`
	RunID         string      `json:"runid"`
}

type PendingRun struct {
	RunID         string    `json:"runid"`
	TestID        string    `json:"testid"`
	App           string    `json:"app"`
	Space         string    `json:"space"`
	Job           string    `json:"job"`
	Jobspace      string    `json:"jobspace"`
	Image         string    `json:"image"`
	Overallstatus string    `json:"overallstatus"`
	Timeout       int       `json:"timeout"`
	RunOn         time.Time `json:"run_on"`
}

type PendingCronRun struct {
	RunID         string      `json:"runid"`
	TestID        string      `json:"testid"`
	CronID        string      `json:"cronid"`
	App           string      `json:"app"`
	Space         string      `json:"space"`
	Job           string      `json:"job"`
	Jobspace      string      `json:"jobspace"`
	Image         string      `json:"image"`
	Overallstatus string      `json:"overallstatus"`
	StartTime     time.Time   `json:"starttime"`
	EndTime       pq.NullTime `json:"endtime,omitempty"`
}
