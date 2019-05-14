package models

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/keremk/challenge-bot/config"
	"github.com/keremk/challenge-bot/db"
	"github.com/keremk/challenge-bot/util"
)

type Challenge struct {
	ID              string
	Name            string
	GithubOwner     string
	GithubOrg       string
	TemplateRepo    string
	CreatedByTeamID string
}

func NewChallenge(input map[string]string) Challenge {
	return Challenge{
		ID:              util.RandomString(16),
		Name:            input["challenge_name"],
		GithubOwner:     input["github_owner"],
		GithubOrg:       input["github_org"],
		TemplateRepo:    input["template_repo"],
		CreatedByTeamID: input["team_id"],
	}
}

func GetChallenge(env config.Environment, name string) (Challenge, error) {
	challenge := Challenge{}
	store, err := db.NewStore(env, db.SettingsCollection)
	if err != nil {
		return challenge, err
	}

	err = store.FindFirst("Name", name, &challenge)
	return challenge, err
}

func GetAllChallenges(env config.Environment) ([]Challenge, error) {
	store, err := db.NewStore(env, db.SettingsCollection)
	if err != nil {
		return nil, err
	}

	var all []Challenge
	result, err := store.FindAll(reflect.TypeOf(all))
	all, ok := result.([]Challenge)
	if !ok {
		return nil, errors.New("[ERROR] Cannot convert")
	}
	return all, err
}

func UpdateChallenge(env config.Environment, challenge Challenge) error {
	store, err := db.NewStore(env, db.SettingsCollection)
	if err != nil {
		return err
	}

	return store.Update(challenge.ID, challenge)
}

func (s Challenge) OrgOrOwner() string {
	if s.GithubOrg != "" {
		return s.GithubOrg
	} else {
		return s.GithubOwner
	}
}

func (s Challenge) TemplateRepositoryURL() string {
	return fmt.Sprintf("https://github.com/%v/%v.git", s.OrgOrOwner(), s.TemplateRepo)
}

func (s Challenge) TrackingIssuesURL() string {
	return fmt.Sprintf("https://github.com/%v/%v/issues", s.OrgOrOwner(), s.TemplateRepo)
}