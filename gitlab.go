package git

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xanzy/go-gitlab"
)

var GitlabServer gitlabServer

type fileContentInter interface {
	RenderYaml() ([]byte, error)
}

type gitlabServer struct {
	Client      *gitlab.Client
	GroupId     *int
	GroupName   string
	ProjectName string
}

// InitGitlabServer init gitlab
func InitGitlabServer(token, url string) error {
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(url))
	if err != nil {
		return err
	}
	GitlabServer.Client = client
	return nil
}

// CreateProject Create a new project
func (git *gitlabServer) CreateProject() (string, error) {
	p := &gitlab.CreateProjectOptions{
		NamespaceID:          git.GroupId,
		Name:                 gitlab.String(git.ProjectName),
		Description:          gitlab.String("kubernetes runtime resource manifests"),
		MergeRequestsEnabled: gitlab.Bool(true),
		SnippetsEnabled:      gitlab.Bool(true),
		Visibility:           gitlab.Visibility(gitlab.PrivateVisibility),
	}
	project, _, err := git.Client.Projects.CreateProject(p)
	if err != nil {
		return fmt.Sprintf("create project: <%v> error", git.ProjectName), err
	}
	return fmt.Sprintf("create project: <%v> ok, project_id: %d", git.ProjectName, project.ID), nil
}

// ListProjectHook list a project's hook
func (git *gitlabServer) ListProjectHook() (data []map[string]interface{}, err error) {
	repoInfo, err := git.GetProject()
	if err != nil {
		return
	}
	repoId := repoInfo["id"]
	p := &gitlab.ListProjectHooksOptions{}
	projectHooks, _, err := git.Client.Projects.ListProjectHooks(repoId, p)
	if err != nil {
		return
	}
	bytes, err := json.Marshal(&projectHooks)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return
	}
	return
}

// IsProjectHookExists if project hook exists return true, otherwise return false
func (git *gitlabServer) IsProjectHookExists(url string) (string, error) {
	projectHookSlice, err := git.ListProjectHook()
	if err != nil {
		return fmt.Sprintf("list project hook: <%v> error", git.ProjectName), err
	}
	for _, m := range projectHookSlice {
		fmt.Println(m["url"], url)
		if m["url"].(string) == url {
			return fmt.Sprintf("project %s hook already exists", git.ProjectName), nil
		}
	}
	return "", errors.New("not found")
}

// CreateProjectHookByPush create a project's push hook
func (git *gitlabServer) CreateProjectHookByPush(url, branch string, pushEvents, enableSSLVerification bool) (string, error) {
	repoInfo, err := git.GetProject()
	if err != nil {
		return "", err
	}
	repoId := repoInfo["id"].(float64)

	_, err = git.IsProjectHookExists(url)
	if err == nil {
		return "", errors.New(fmt.Sprintf("url: %s already exists", url))
	}
	p := &gitlab.AddProjectHookOptions{
		URL:                    &url,
		PushEventsBranchFilter: &branch,
		PushEvents:             &pushEvents,
		EnableSSLVerification:  &enableSSLVerification,
	}
	projectHooks, _, err := git.Client.Projects.AddProjectHook(int(repoId), p)
	if err != nil {
		return fmt.Sprintf("add project hook: <%v> error", git.ProjectName), err
	}
	return fmt.Sprintf("add project hook: <%v> ok, hook_id: %d", git.ProjectName, projectHooks.ID), nil
}

// CreateProjectHookByTag create a project's tag hook
func (git *gitlabServer) CreateProjectHookByTag(url, branch string, tagPushEvents, enableSSLVerification bool) (string, error) {
	repoInfo, err := git.GetProject()
	if err != nil {
		return "", err
	}
	repoId := repoInfo["id"].(float64)

	_, err = git.IsProjectHookExists(url)
	if err == nil {
		return "", errors.New(fmt.Sprintf("url: %s already exists", url))
	}
	p := &gitlab.AddProjectHookOptions{
		URL:                    &url,
		PushEventsBranchFilter: &branch,
		TagPushEvents:          &tagPushEvents,
		EnableSSLVerification:  &enableSSLVerification,
	}
	projectHooks, _, err := git.Client.Projects.AddProjectHook(int(repoId), p)
	if err != nil {
		return fmt.Sprintf("add project hook: <%v> error", git.ProjectName), err
	}
	return fmt.Sprintf("add project hook: <%v> ok, hook_id: %d", git.ProjectName, projectHooks.ID), nil
}

// ListProject list all repo by group
func (git *gitlabServer) ListProject() ([]map[string]interface{}, error) {
	var (
		simple = true
		data   []map[string]interface{}
	)
	lp := &gitlab.ListGroupProjectsOptions{
		Simple: &simple,
	}
	projectGroup, _, err := git.Client.Groups.ListGroupProjects(*git.GroupId, lp)
	if err != nil {
		return data, err
	}
	bytes, err := json.Marshal(&projectGroup)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return data, err
	}
	return data, nil
}

// GetProject get project info
func (git *gitlabServer) GetProject() (map[string]interface{}, error) {
	data := make(map[string]interface{})
	repoSlice, err := git.ListProject()
	if err != nil {
		return data, err
	}
	for _, project := range repoSlice {
		if project["name"] == git.ProjectName {
			return project, nil
		}
	}
	return data, errors.New("not found")
}

// getProjectPath get project path
func (git *gitlabServer) getProjectPath() string {
	return fmt.Sprintf("%s/%s", git.GroupName, git.ProjectName)
}

// GetProjectId if project exists return (projectId, true), otherwise return (0, false)
func (git *gitlabServer) GetProjectId() (float64, error) {
	repoSlice, err := git.ListProject()
	if err != nil {
		return 0, err
	}
	for _, project := range repoSlice {
		if project["name"] == git.ProjectName {
			return project["id"].(float64), nil
		}
	}
	return 0, errors.New(fmt.Sprintf("project %s not exists", git.ProjectName))
}

// IsProjectExists if repo exists return true, otherwise return false
func (git *gitlabServer) IsProjectExists() (string, error) {
	repoSlice, err := git.ListProject()
	if err != nil {
		return "", err
	}
	for _, project := range repoSlice {
		if project["name"] == git.ProjectName {
			return fmt.Sprintf("project name %s already exists", git.ProjectName), nil
		}
	}
	return "", errors.New("not found")
}

// ListProjectCommit Get a list of repository commits in a project.
func (git *gitlabServer) ListProjectCommit(branch string) (data []map[string]interface{}, err error) {
	projectId, err := git.GetProjectId()
	if err != nil {
		return
	}
	options := &gitlab.ListCommitsOptions{
		RefName: &branch,
	}

	commitSlice, _, err := git.Client.Commits.ListCommits(int(projectId), options)
	if err != nil {
		return
	}
	bytes, err := json.Marshal(&commitSlice)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return
	}
	return
}

// ListProjectCommitFormat Get a list of repository commits in a project.
func (git *gitlabServer) ListProjectCommitFormat(branch string) (data []map[string]interface{}, err error) {
	projectId, err := git.GetProjectId()
	if err != nil {
		return
	}
	options := &gitlab.ListCommitsOptions{
		RefName: &branch,
	}

	commitSlice, _, err := git.Client.Commits.ListCommits(int(projectId), options)
	if err != nil {
		return
	}
	for _, commit := range commitSlice {
		opt := make(map[string]interface{})
		opt["commit_id"] = commit.ShortID
		opt["commit_message"] = commit.Title
		opt["commit_author"] = commit.AuthorName
		data = append(data, opt)
	}

	return
}

// RollbackProjectCommit Reverts a commit in a given branch
func (git *gitlabServer) RollbackProjectCommit(branch, commitId string) (string, error) {
	projectId, err := git.GetProjectId()
	if err != nil {
		return "get project id error", err
	}
	options := &gitlab.RevertCommitOptions{
		Branch: &branch,
	}
	commit, _, err := git.Client.Commits.RevertCommit(int(projectId), commitId, options)
	if err != nil {
		return fmt.Sprintf("rollback commit %s/%s error", branch, commitId), err
	}
	fmt.Printf("commit: %v\n", commit)
	return fmt.Sprintf("rollback commit %s/%s ok", branch, commitId), nil
}

// CreateFile Create a new repository file
func (git *gitlabServer) CreateFile(branch, filename, fileContent, commitMessage string) (string, error) {
	cf := &gitlab.CreateFileOptions{
		Branch:        gitlab.String(branch),
		Content:       gitlab.String(fileContent),
		CommitMessage: gitlab.String(commitMessage),
	}
	_, resp, err := git.Client.RepositoryFiles.CreateFile(git.getProjectPath(), filename, cf)
	if err != nil {
		return fmt.Sprintf("create file: <%s> error, err: %v\n", filename, resp.Response.Status), err
	}
	return fmt.Sprintf("create file: <%s> ok", filename), err
}

// CreateFileInter Create a new repository file
func (git *gitlabServer) CreateFileInter(branch, filename string, f fileContentInter, commitMessage string) (string, error) {
	bytes, err := f.RenderYaml()
	if err != nil {
		return "", errors.New(fmt.Sprintf("renderYaml interface err: %v", err))
	}
	cf := &gitlab.CreateFileOptions{
		Branch:        gitlab.String(branch),
		Content:       gitlab.String(string(bytes)),
		CommitMessage: gitlab.String(commitMessage),
	}
	_, resp, err := git.Client.RepositoryFiles.CreateFile(git.getProjectPath(), filename, cf)
	if err != nil {
		return fmt.Sprintf("create file: <%s> error, err: %v\n", filename, resp.Response.Status), err
	}
	return fmt.Sprintf("create file: <%s> ok", filename), err
}

// UpdateFileInter Update a repository file
func (git *gitlabServer) UpdateFileInter(branch, filename string, f fileContentInter, commitMessage string) (string, error) {
	bytes, err := f.RenderYaml()
	if err != nil {
		return "", errors.New(fmt.Sprintf("renderYaml interface err: %v", err))
	}
	uf := &gitlab.UpdateFileOptions{
		Branch:        gitlab.String(branch),
		Content:       gitlab.String(string(bytes)),
		CommitMessage: gitlab.String(commitMessage),
	}
	_, resp, err := git.Client.RepositoryFiles.UpdateFile(git.getProjectPath(), filename, uf)
	if err != nil {
		return fmt.Sprintf("update file: <%s> error, err: %v\n", filename, resp.Response.Status), err
	}
	return fmt.Sprintf("update file: <%s> ok", filename), err
}

// UpdateFile Update a repository file
func (git *gitlabServer) UpdateFile(branch, filename, fileContent, commitMessage string) (string, error) {
	uf := &gitlab.UpdateFileOptions{
		Branch:        gitlab.String(branch),
		Content:       gitlab.String(fileContent),
		CommitMessage: gitlab.String(commitMessage),
	}
	_, resp, err := git.Client.RepositoryFiles.UpdateFile(git.getProjectPath(), filename, uf)
	if err != nil {
		return fmt.Sprintf("update file: <%s> error, err: %v\n", filename, resp.Response.Status), err
	}
	return fmt.Sprintf("update file: <%s> ok", filename), err
}

// GetRawFile get a file content
func (git *gitlabServer) GetRawFile(branch, filename string) (string, error) {
	gf := &gitlab.GetRawFileOptions{
		Ref: gitlab.String(branch),
	}
	body, resp, err := git.Client.RepositoryFiles.GetRawFile(git.getProjectPath(), filename, gf)
	if err != nil {
		return fmt.Sprintf("get file: <%s> error, err: %v\n", filename, resp.Response.Status), err
	}
	return string(body), nil
}

// IsFileExists if file exists return true, otherwise return false
func (git *gitlabServer) IsFileExists(branch, filename string) bool {
	gf := &gitlab.GetFileOptions{
		Ref: gitlab.String(branch),
	}
	_, _, err := git.Client.RepositoryFiles.GetFile(git.getProjectPath(), filename, gf)
	if err != nil {
		return false
	}
	return true
}

// CreateTag create a new tag
func (git *gitlabServer) CreateTag(branch, tagName, message string) error {
	projectId, err := git.GetProjectId()
	if err != nil {
		return err
	}
	options := &gitlab.CreateTagOptions{
		TagName: &tagName,
		Ref:     &branch,
		Message: &message,
	}

	tag, _, err := git.Client.Tags.CreateTag(int(projectId), options)
	if err != nil {
		return err
	}
	fmt.Printf("tag: %v", tag)
	return nil
}
