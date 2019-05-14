package db

import (
	"errors"
	"reflect"

	"github.com/keremk/challenge-bot/config"
)

const SlackUsersCollection = "slackusers"
const SlackTeamsCollection = "slackteams"
const SettingsCollection = "challengesettings"

type CrudOps interface {
	Update(key string, obj interface{}) error
	FindByID(id string, obj interface{}) error
	FindFirst(key, value string, obj interface{}) error
	FindAll(itemType reflect.Type) (interface{}, error)
}

func NewStore(env config.Environment, collection string) (CrudOps, error) {
	switch env.DbProvider {
	case Firestore:
		return FirestoreDb{
			env:        env,
			collection: collection,
		}, nil
	default:
		return nil, errors.New("[ERROR] db provider not known or unspecified")
	}
}