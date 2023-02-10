package youtrackApi2023

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"log"
	"net/url"
)

type ProjectID struct {
	ID string `json:"id"`
}

type Reporter struct {
	FullName string `json:"fullName"`
}

type CustomFieldAPI struct {
	Name string `json:"name"`
	Type string `json:"$type"`
	//Value string `json:"value"`
	Value json.RawMessage `json:"value"`
}

// One record of history of an issue
type IssueHistoryItem struct {
	Timestamp int64 `json:"timestamp"`
	Added     []IssueHistoryItemAdded
	Removed   []IssueHistoryItemRemoved
	Author    IssueHistoryItemAuthor
	Field     IssueHistoryItemField
	Type      string `json:"$type"`
}

type IssueHistoryItemField struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Type  string `json:"$type"`
}

type IssueHistoryItemAdded struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Type  string `json:"$type"`
}

type IssueHistoryItemRemoved struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Type  string `json:"$type"`
}

type IssueHistoryItemAuthor struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Type  string `json:"$type"`
}

type Issue struct {
	Summary      string           `json:"summary"`
	Description  string           `json:"description"`
	Project      ProjectID        `json:"project"`
	ID           string           `json:"id"`
	Type         string           `json:"$type"`
	IdReadable   string           `json:"idReadable"`
	CreatedBy    Reporter         `json:"reporter"`
	CustomFields []CustomFieldAPI `json:"customFields"`
}

type IDResult struct {
	ID string `json:"id"`
}

type IssueResult struct {
	IDResult
	NumberInProject int `json:"numberInProject"`
}

// Custom field with formatted structure name, value
type CustomFieldFormatted struct {
	Name  string
	Value string
}

type TextIssueCustomField struct {
	Text string `json:"text"`
	Type string `json:"$type"`
}

type SingleEnumIssueCustomField struct {
	Name string `json:"name"`
}

type SingleVersionIssueCustomField struct {
	Name string `json:"name"`
}

type PeriodIssueCustomField struct {
	Presentation string `json:"presentation"`
}

type StateMachineIssueCustomField struct {
	Name string `json:"name"`
}

type CustomFieldsFormattedSlice []CustomFieldFormatted

type FieldNotFoundError struct{}

func (m *FieldNotFoundError) Error() string {
	return "FieldNotFound"
}

// returns issue history
//
// warning, history of text fields is not yet supported - it is not held in structure common for added but only a string currently added is in api return structure :(
func (api *Api) GetIssueHistory(ctx context.Context, idReadable string) ([]IssueHistoryItem, error) {

	var records []IssueHistoryItem

	u := &url.URL{
		Path:     "issues/" + idReadable + "/activities",
		RawQuery: "categories=CustomFieldCategory&fields=author(name,login),timestamp,added(name,login),removed(name,login),field(id,name,text)",
	}
	err := api.Get(ctx, u, &records)
	/*
		if err != nil {
			// WARNING, text updates are not yet supported - for example when field is a textfile and added does not contain structure but only that string(!)
			//return nil, err

		}
	*/

	return records, err
}

// Tries to parse resolved date of item from history records
func (Issue) GetResolvedDateTimestamp(historyItems []IssueHistoryItem) int64 {
	var result int64
	result = -1

	if len(historyItems) > 0 {
		for i := len(historyItems) - 1; i >= 0; i-- {
			item := historyItems[i]
			if item.Field.Name == "State" && item.Added != nil && item.Added[0].Name == "Available" {
				result = item.Timestamp
				break
			}

		}
	}

	return result
}

// ListIssues returns a list of Projects the user has access to.
// For more information, see
// https://www.jetbrains.com/help/youtrack/incloud/2019.3/resource-api-admin-projects.html
func (api *Api) ListIssues(ctx context.Context, query string) ([]Issue, error) {
	var Issues []Issue
	u := &url.URL{
		Path:     "issues",
		RawQuery: "fields=id,description,summary,idReadable,created,updated,resolved,reporter(fullName),updater(fullName),customFields(name,value(name,text,presentation,fullName,color(background,foreground)))&query=" + query,
	}
	err := api.Get(ctx, u, &Issues)
	if err != nil {
		return nil, err
	}
	return Issues, nil
}

func (c CustomFieldsFormattedSlice) FindCustomField(name string) (string, error) {

	for _, t := range c {
		if t.Name == name {
			return t.Value, nil
		}
	}

	return "", &FieldNotFoundError{}
}

// ignore missing columns, just return empty string
func (c CustomFieldsFormattedSlice) FindCustomFieldNoError(name string) string {

	for _, t := range c {
		if t.Name == name {
			return t.Value
		}
	}

	return ""
}

// Parse custom fields retrieved from Youtrack api and creates easy structure with name, value pair
func (issue *Issue) ParseCustomFields() CustomFieldsFormattedSlice {
	var fields []CustomFieldFormatted

	if len(issue.CustomFields) > 0 {
		for id := 0; id < len(issue.CustomFields); id++ {
			f := issue.CustomFields[id]

			switch f.Type {
			case "TextIssueCustomField":

				var x TextIssueCustomField
				if err := json.Unmarshal(f.Value, &x); err == nil {
					//fmt.Printf(" -> TextIssueCustomField ["+f.Name+"] => [%+v] \n", x.Text)

					item := CustomFieldFormatted{f.Name, x.Text}
					fields = append(fields, item)
				}

			case "SingleEnumIssueCustomField":

				var x SingleEnumIssueCustomField
				if err := json.Unmarshal(f.Value, &x); err == nil {
					//fmt.Printf(" -> SingleEnumIssueCustomField ["+f.Name+"] => [%+v] \n", x.Name)
					item := CustomFieldFormatted{f.Name, x.Name}
					fields = append(fields, item)
				}

			case "SimpleIssueCustomField":

				var result string = ""
				if err := json.Unmarshal(f.Value, &result); err == nil {

					//fmt.Printf(" -> SimpleIssueCustomField ["+f.Name+"] => [%+v] \n", result)
					item := CustomFieldFormatted{f.Name, result}
					fields = append(fields, item)
				}

			case "DateIssueCustomField":

				var result string = ""
				if err := json.Unmarshal(f.Value, &result); err == nil {

					//fmt.Printf(" -> DateIssueCustomField ["+f.Name+"] => [%+v] \n", result)
					item := CustomFieldFormatted{f.Name, result}
					fields = append(fields, item)
				}

			case "SingleVersionIssueCustomField":

				var x SingleEnumIssueCustomField
				if err := json.Unmarshal(f.Value, &x); err == nil {
					//fmt.Printf(" -> SingleVersionIssueCustomField ["+f.Name+"] => [%+v] \n", x.Name)
					item := CustomFieldFormatted{f.Name, x.Name}
					fields = append(fields, item)
				}

			case "StateMachineIssueCustomField":
				var x StateMachineIssueCustomField
				if err := json.Unmarshal(f.Value, &x); err == nil {
					//fmt.Printf(" -> SingleVersionIssueCustomField ["+f.Name+"] => [%+v] \n", x.Name)
					item := CustomFieldFormatted{f.Name, x.Name}
					fields = append(fields, item)
				}

			case "PeriodIssueCustomField":

				var x PeriodIssueCustomField
				if err := json.Unmarshal(f.Value, &x); err == nil {
					//fmt.Printf(" -> PeriodIssueIssueCustomField ["+f.Name+"] => [%+v] \n", x.Presentation)
					item := CustomFieldFormatted{f.Name, x.Presentation}
					fields = append(fields, item)
				}

			default:
				fmt.Println(" .> [" + f.Name + "], f.Type " + f.Type + " is not supported yet.")
			}

		}
	}

	return fields
}

// IssueURL returns a user facing (rather than REST API) URL to the issue.
// Note that because this issue uses the short project name in the URL, the link
// could be broken if the project short name changes.
func IssueURL(baseURL *url.URL, shortProjectName string, issueNumberInProject int) *url.URL {
	path := fmt.Sprintf("../issue/%s-%d", shortProjectName, issueNumberInProject)
	return baseURL.ResolveReference(&url.URL{Path: path})
}

func (api *Api) IssueURL(shortProjectName string, issueNumberInProject int) *url.URL {
	return IssueURL(api.BaseURL, shortProjectName, issueNumberInProject)
}

// CreateIssue returns the issue ID on success.
func (api *Api) CreateIssue(ctx context.Context, project, summary, description string) (*IssueResult, error) {
	issue := &Issue{
		Summary:     summary,
		Description: description,
		Project: ProjectID{
			ID: project,
		},
	}

	result := new(IssueResult)
	u := &url.URL{
		Path:     "issues",
		RawQuery: "fields=id,numberInProject",
	}

	err := api.Post(ctx, u, issue, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type IssueAttachment struct {
	Name          string `json:"name"`
	Base64Content string `json:"base64Content"`
}

// CreateIssueAttachment attached a file to the given issue ID. On success, the attachment ID is returned.
func (api *Api) CreateIssueAttachment(ctx context.Context, issueID string, attachment io.Reader, name, mediaType string) (string, error) {

	data, err := ioutil.ReadAll(attachment)
	if err != nil {
		log.Print("Failed to read attachment", err)
		return "", err
	}

	issueAttachment := &IssueAttachment{
		Name:          name,
		Base64Content: "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}

	u := &url.URL{
		Path: fmt.Sprintf("issues/%s/attachments", issueID),
	}
	result := &IDResult{}

	err = api.Post(ctx, u, issueAttachment, result)
	if err != nil {
		log.Print("Failed to post attachment.", err)
		return "", err
	}

	return result.ID, nil
}
