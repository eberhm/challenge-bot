package slack

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/keremk/challenge-bot/models"

	"github.com/keremk/challenge-bot/config"
	slackApi "github.com/nlopes/slack"
)

type command struct {
	mainCommand  string
	subCommand   string
	arg          string
	slashCommand *slackApi.SlashCommand
}

func ExecuteCommand(env config.Environment, request *http.Request) error {
	slashCommand, err := parsePayload(request, env.VerificationToken)
	if err != nil {
		return err
	}

	c := parseSlashCommand(slashCommand)
	log.Println("[INFO] Challenge command")
	log.Println("[INFO] Main Command", c.mainCommand)
	log.Println("[INFO] Sub Command", c.subCommand)
	log.Println("[INFO] Text", c.arg)

	switch c.mainCommand {
	case "/challenge":
		log.Println("[INFO] Challenge command is invoked")
		fallthrough
	case "/challengetest":
		switch c.subCommand {
		case "help":
			log.Println("[INFO] HELP is called here")
			go executeHelp(c)
		case "new":
			go executeNewChallenge(env, c)
		case "send":
			go executeSendChallenge(env, c)
		}
	default:
		log.Println("[ERROR] Unexpected Command ", c.mainCommand)
		return errors.New("Unexpected command")
	}
	return nil
}

func parsePayload(request *http.Request, verificationToken string) (*slackApi.SlashCommand, error) {
	s, err := slackApi.SlashCommandParse(request)
	if err != nil {
		log.Println("[ERROR] Unable to parse command ", err)
		return nil, err
	}

	if !s.ValidateToken(verificationToken) {
		log.Println("[ERROR] Unable to validate command ", err)
		return nil, ValidationError{}
	}
	return &s, nil
}

func parseSlashCommand(slashCommand *slackApi.SlashCommand) command {
	helpCommand := command{
		mainCommand:  slashCommand.Command,
		subCommand:   "help",
		arg:          "",
		slashCommand: slashCommand,
	}

	if len(slashCommand.Text) == 0 {
		return helpCommand
	}

	c := strings.Split(slashCommand.Text, " ")
	switch len(c) {
	case 1:
		return command{
			mainCommand:  slashCommand.Command,
			subCommand:   c[0],
			arg:          "",
			slashCommand: slashCommand,
		}
	case 2:
		return command{
			mainCommand:  slashCommand.Command,
			subCommand:   c[0],
			arg:          c[1],
			slashCommand: slashCommand,
		}
	default:
		return helpCommand
	}
}

func executeHelp(c command) error {
	helpText := `
{
	"blocks": [
		{
			"type": "section", 
			"text": {
				"type": "mrkdwn",
				"text": "Hello and welcome to the coding challenge tool. You can use the following commands:"
			} 
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "*/challenge help* : Displays this message"
			}
		}, 
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "*/challenge new* : Opens a dialog to create a new challenge"
			}
		},
		{
			"type": "section",
			"text": {
				"type": "mrkdwn",
				"text": "*/challenge send* : Opens a dialog to send a challenge to a candidate"
			}
		}
	]
}
`
	err := sendDelayedResponse(c.slashCommand.ResponseURL, helpText)
	return err
}

func executeSendChallenge(env config.Environment, c command) error {
	s := c.slashCommand
	token, err := getBotToken(env, s.TeamID)
	if err != nil {
		return err
	}

	// Create the dialog and send a message to open it
	state := dialogState{
		channelID:    s.ChannelID,
		settingsName: c.arg,
	}
	challengeList, err := challengeNames(env)
	if err != nil {
		return err
	}
	dialog := sendChallengeDialog(s.TriggerID, state, challengeList)

	slackClient := slackApi.New(token)
	err = slackClient.OpenDialog(s.TriggerID, *dialog)
	if err != nil {
		log.Println("[ERROR] Cannot create the dialog ", err)
	}
	return err
}

func executeNewChallenge(env config.Environment, c command) error {
	s := c.slashCommand
	token, err := getBotToken(env, s.TeamID)
	if err != nil {
		return err
	}

	// Create the dialog and send a message to open it
	state := dialogState{
		channelID:    s.ChannelID,
		settingsName: c.arg,
	}

	dialog := newChallengeDialog(s.TriggerID, state)

	slackClient := slackApi.New(token)
	err = slackClient.OpenDialog(s.TriggerID, *dialog)
	if err != nil {
		log.Println("[ERROR] Cannot create the dialog ", err)
	}
	return err

	return nil
}

func challengeNames(env config.Environment) ([]string, error) {
	settings, err := models.GetAllChallenges(env)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, setting := range settings {
		names = append(names, setting.Name)
	}
	return names, nil
}

func sendChallengeDialog(triggerID string, state dialogState, options []string) *slackApi.Dialog {
	candidateNameElement := slackApi.NewTextInput("candidate_name", "Candidate Name", "")
	githubNameElement := slackApi.NewTextInput("github_alias", "Github Alias", "")
	resumeURLElement := slackApi.NewTextInput("resume_URL", "Resume URL", "")
	selectOptions := make([]slackApi.DialogSelectOption, len(options))
	for i, v := range options {
		selectOptions[i] = slackApi.DialogSelectOption{
			Label: v,
			Value: v,
		}
	}
	challengeNameElement := slackApi.NewStaticSelectDialogInput("challenge_name", "Challenge Name", selectOptions)

	elements := []slackApi.DialogElement{
		candidateNameElement,
		githubNameElement,
		resumeURLElement,
		challengeNameElement,
	}

	return &slackApi.Dialog{
		TriggerID:      triggerID,
		CallbackID:     "send_challenge",
		Title:          "Send Coding Challenge",
		SubmitLabel:    "Send",
		NotifyOnCancel: false,
		State:          state.string(),
		Elements:       elements,
	}
}

func newChallengeDialog(triggerID string, state dialogState) *slackApi.Dialog {
	githubOrgElement := slackApi.NewTextInput("github_org", "Github Organization", "")
	githubOwnerElement := slackApi.NewTextInput("github_owner", "Github Owner", "")
	challengeNameElement := slackApi.NewTextInput("challenge_name", "Challenge Name", "")
	templateRepoName := slackApi.NewTextInput("template_repo", "Template Repo Name", "")
	elements := []slackApi.DialogElement{
		githubOrgElement,
		githubOwnerElement,
		challengeNameElement,
		templateRepoName,
	}
	return &slackApi.Dialog{
		TriggerID:      triggerID,
		CallbackID:     "new_challenge",
		Title:          "New Coding Challenge",
		SubmitLabel:    "Create",
		NotifyOnCancel: false,
		State:          state.string(),
		Elements:       elements,
	}
}