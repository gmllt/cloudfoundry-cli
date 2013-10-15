package user

import (
	"cf/api"
	"cf/requirements"
	"cf/terminal"
	"errors"
	"github.com/codegangsta/cli"
)

type DeleteUser struct {
	ui       terminal.UI
	userRepo api.UserRepository
}

func NewDeleteUser(ui terminal.UI, userRepo api.UserRepository) (cmd DeleteUser) {
	cmd.ui = ui
	cmd.userRepo = userRepo
	return
}

func (cmd DeleteUser) GetRequirements(reqFactory requirements.Factory, c *cli.Context) (reqs []requirements.Requirement, err error) {
	if len(c.Args()) != 1 {
		err = errors.New("Invalid usage")
		cmd.ui.FailWithUsage(c, "delete-user")
		return
	}

	reqs = append(reqs, reqFactory.NewLoginRequirement())

	return
}

func (cmd DeleteUser) Run(c *cli.Context) {
	username := c.Args()[0]
	force := c.Bool("f")

	if !force && !cmd.ui.Confirm("Really delete user %s?%s",
		terminal.EntityNameColor(username),
		terminal.PromptColor(">"),
	) {
		return
	}

	cmd.ui.Say("Deleting user %s...", terminal.EntityNameColor(username))

	user, apiResponse := cmd.userRepo.FindByUsername(username)
	if apiResponse.IsError() {
		cmd.ui.Failed(apiResponse.Message)
		return
	}
	if apiResponse.IsNotFound() {
		cmd.ui.Ok()
		cmd.ui.Warn("User %s does not exist.", username)
		return
	}

	apiResponse = cmd.userRepo.Delete(user)
	if apiResponse.IsNotSuccessful() {
		cmd.ui.Failed(apiResponse.Message)
		return
	}

	cmd.ui.Ok()
}
