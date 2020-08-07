package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/build"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/pipelines"
	"github.com/microsoft/azure-devops-go-api/azuredevops/release"
	"github.com/microsoft/azure-devops-go-api/azuredevops/taskagent"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func getDetails(ctx context.Context, connection *azuredevops.Connection) error {
	// Create a client to interact with the Core area
	coreClient, err := core.NewClient(ctx, connection)
	if err != nil {
		log.Fatal(err)
	}

	// Get first page of the list of team projects for your organization
	responseValue, err := coreClient.GetProjects(ctx, core.GetProjectsArgs{})
	if err != nil {
		log.Fatal(err)
	}

	index := 0
	for responseValue != nil {
		// Log the page of team project names
		for _, teamProjectReference := range (*responseValue).Value {
			log.Printf("Name[%v] = %v", index, *teamProjectReference.Name)
			index++
		}

		// if continuationToken has a value, then there is at least one more page of projects to get
		if responseValue.ContinuationToken != "" {
			// Get next page of team projects
			projectArgs := core.GetProjectsArgs{
				ContinuationToken: &responseValue.ContinuationToken,
			}
			responseValue, err = coreClient.GetProjects(ctx, projectArgs)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			responseValue = nil
		}
	}

	return nil
}

func listPipelines(ctx context.Context, connection *azuredevops.Connection) ([]pipelines.Pipeline, error) {
	pipelineClient := pipelines.NewClient(ctx, connection)

	projectName := "testken"

	pipelineList, err := pipelineClient.ListPipelines(ctx, pipelines.ListPipelinesArgs{Project: &projectName})
	if err != nil {
		log.Fatal(err)
	}

	return pipelineList.Value, nil
}

func getPipeline(ctx context.Context, connection *azuredevops.Connection, pipelineId int) (string, error) {
	pipelineClient := pipelines.NewClient(ctx, connection)

	projectName := "testken"

	pipelineDetails, err := pipelineClient.GetPipeline(ctx, pipelines.GetPipelineArgs{Project: &projectName, PipelineId: &pipelineId})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("pipeline details name: %s\n", pipelineDetails.Name)
	fmt.Printf("pipeline details config: %s\n", pipelineDetails.Links)

	// find self link.... has magic json that shows the build pipeline :)
	m := pipelineDetails.Links.(map[string]interface{})
	a := m["self"].(map[string]interface{})
	buildURL := a["href"].(string)
	fmt.Printf("URL is %s\n", buildURL)

	req, _ := http.NewRequest("GET", buildURL, nil)
	client := http.Client{}
	req.Header.Add("Authorization", connection.AuthorizationString)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("BOOM %s\n", err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	strBody := string(body)
	return strBody, nil
}

func createPipeline(ctx context.Context, connection *azuredevops.Connection, projectName string) error {
	pipelineClient := pipelines.NewClient(ctx, connection)

	config := pipelines.CreatePipelineParameters{}
	pipelineName := "imported pipeline"
	config.Name = &pipelineName
	folder := "unsure"
	config.Folder = &folder
	config.Configuration = &pipelines.CreatePipelineConfigurationParameters{}

	pl, err := pipelineClient.CreatePipeline(ctx, pipelines.CreatePipelineArgs{Project: &projectName, InputParameters: &config})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("pipeline created %v\n", pl)

	return nil
}

func getPipelineREST(connection *azuredevops.Connection, org string, projectName string, definitionID int) error {

	urlTemplate := `GET https://dev.azure.com/%s/%s/_apis/build/definitions/%d?api-version=6.0-preview.7`
	url := fmt.Sprintf(urlTemplate, org, projectName, definitionID)

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", connection.AuthorizationString)
	req.Header.Add("Content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error on put %s\n", err.Error())
		panic(err)
	}

	fmt.Printf("status code is %d\n", resp.StatusCode)
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("body is %s\n", string(b))
	return nil
}

func createPipelineREST(ctx context.Context, connection *azuredevops.Connection, org string, projectName string, body string) error {

	urlTemplate := `https://dev.azure.com/%s/%s/_apis/build/definitions?api-version=6.0-preview.7`
	url := fmt.Sprintf(urlTemplate, org, projectName)

	// utter hack to see if this works.

	newBody := strings.ReplaceAll(body, "testken-ASP.NET Core-CI", "newpipeline")
	newBody = strings.ReplaceAll(newBody, "https://kenfaulkner.visualstudio.com/_apis/projects/7f6695b5-d23e-4e0e-91eb-567124a01d80", "")
	newBody = strings.ReplaceAll(newBody, "https://kenfaulkner.visualstudio.com/7f6695b5-d23e-4e0e-91eb-567124a01d80/_apis/", "")

	client := &http.Client{}

	req, err := http.NewRequest("POST", url, strings.NewReader(newBody))
	req.Header.Add("Authorization", connection.AuthorizationString)
	req.Header.Add("Content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error on put %s\n", err.Error())
		panic(err)
	}

	fmt.Printf("status code is %d\n", resp.StatusCode)
	b, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("body is %s\n", string(b))

	pipelineClient := pipelines.NewClient(ctx, connection)

	config := pipelines.CreatePipelineParameters{}
	pipelineName := "imported pipeline"
	config.Name = &pipelineName
	folder := "unsure"
	config.Folder = &folder
	config.Configuration = &pipelines.CreatePipelineConfigurationParameters{}

	pl, err := pipelineClient.CreatePipeline(ctx, pipelines.CreatePipelineArgs{Project: &projectName, InputParameters: &config})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("pipeline created %v\n", pl)

	return nil
}

func getVariableGroups(ctx context.Context, connection *azuredevops.Connection, projectName string) ([]taskagent.VariableGroup, error) {

	taskClient, err := taskagent.NewClient(ctx, connection)
	if err != nil {
		log.Fatal(err)
	}

	res, err := taskClient.GetVariableGroups(ctx, taskagent.GetVariableGroupsArgs{Project: &projectName})

	for _, r := range *res {
		fmt.Printf("group name %s\n", *r.Name)
		fmt.Printf("group variables %v\n", *r.Variables)
	}

	return *res, nil
}

func getReleases(ctx context.Context, connection *azuredevops.Connection, projectName string) ([]release.ReleaseDefinition, error) {

	releaseClient, err := release.NewClient(ctx, connection)
	if err != nil {
		log.Fatal(err)
	}

	res, err := releaseClient.GetReleaseDefinitions(ctx, release.GetReleaseDefinitionsArgs{Project: &projectName})

	return res.Value, nil

}

func getProjectBuildUrls(ctx context.Context, connection *azuredevops.Connection, projectName string) ([]build.BuildDefinitionReference, error) {

	buildClient, err := build.NewClient(ctx, connection)
	if err != nil {
		log.Fatal(err)
	}

	defResp, err := buildClient.GetDefinitions(ctx, build.GetDefinitionsArgs{Project: &projectName})

	return defResp.Value, nil
}

func writeFile(dir string, filename string, data string) error {

	os.MkdirAll(dir, os.FileMode(600))
	fullPath := fmt.Sprintf("%s%s%s", dir, string(os.PathSeparator), filename)
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	// not checking if all bytes written etc... hack hack hack
	f.WriteString(data)
	f.Close()
	return nil
}

func writeVariableGroupToFile(variableGroups []taskagent.VariableGroup, outputDir string) error {

	for _, vg := range variableGroups {

		data, err := json.Marshal(vg.Variables)
		if err != nil {
			fmt.Printf("cannot marshal variable groups %s\n", err.Error())
			return err
		}

		writeFile(outputDir, "vargroup-"+*vg.Name+".json", string(data))
	}

	return nil
}

func processReleases(ctx context.Context, connection *azuredevops.Connection, projectName string, outputDir string) error {
	releases, err := getReleases(ctx, connection, projectName)
	if err != nil {
		return err
	}

	for _, r := range releases {
		fmt.Printf("res %s\n", r.Name)
		release := httpGet(*r.Url, connection.AuthorizationString)
		writeFile(outputDir, "release-"+*r.Name+".json", release)
	}

	return nil
}

func processVariableGroups(ctx context.Context, connection *azuredevops.Connection, projectName string, outputDir string) error {
	variableGroups, err := getVariableGroups(ctx, connection, projectName)
	if err != nil {
		return err
	}

	err = writeVariableGroupToFile(variableGroups, outputDir)
	if err != nil {
		return err
	}
	return nil
}

func processBuildPipelines(ctx context.Context, connection *azuredevops.Connection, projectName string, outputDir string) error {
	buildDetails, err := getProjectBuildUrls(ctx, connection, projectName)
	if err != nil {
		fmt.Printf("ERROR while getting project urls: %s\n", err.Error())
		return err
	}

	for _, bd := range buildDetails {
		pipelineJSON := httpGet(*bd.Url, connection.AuthorizationString)
		writeFile(outputDir, "build-"+*bd.Name+".json", pipelineJSON)
	}

	return nil
}

func httpGet(url string, auth string) string {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", auth)
	req.Header.Add("Content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("error on put %s\n", err.Error())
		panic(err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	json := string(b)
	return json

}
func main() {

	orgURL := flag.String("orgurl","","organisational URL. eg. https://dev.azure.com/mycompany")
	pat := flag.String("pat","","Personal Access Token. Generate from AzureDevops. Needs read access to build, variable groups and releases")
	projectName := flag.String("projectname","","project name")
	downloadDir := flag.String("output","",`output directory. eg c:\temp\devopsbackup`)

	flag.Parse()

	if *orgURL == "" || *pat == "" || *projectName == "" || *downloadDir == "" {
		fmt.Printf("Need to provide all params\n")
		return
	}

	// Create a connection to your organization
	connection := azuredevops.NewPatConnection(*orgURL, *pat)
	ctx := context.Background()

	processReleases(ctx, connection, *projectName, *downloadDir )
	processVariableGroups(ctx, connection, *projectName, *downloadDir)
	processBuildPipelines(ctx, connection, *projectName, *downloadDir)

}
