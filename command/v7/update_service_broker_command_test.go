package v7_test

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/cli/actor/actionerror"
	"code.cloudfoundry.org/cli/actor/v7action"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/command/commandfakes"
	v7 "code.cloudfoundry.org/cli/command/v7"
	"code.cloudfoundry.org/cli/command/v7/v7fakes"
	"code.cloudfoundry.org/cli/resources"
	"code.cloudfoundry.org/cli/util/configv3"
	"code.cloudfoundry.org/cli/util/ui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("update-service-broker command", func() {
	const (
		binaryName        = "cf-command"
		serviceBrokerName = "fake-service-broker-name"
		username          = "fake-username"
		password          = "fake-password"
		url               = "fake-url"
	)

	var (
		cmd                          *v7.UpdateServiceBrokerCommand
		fakeUpdateServiceBrokerActor *v7fakes.FakeActor
		fakeSharedActor              *commandfakes.FakeSharedActor
		fakeConfig                   *commandfakes.FakeConfig
		input                        *Buffer
		testUI                       *ui.UI
	)

	BeforeEach(func() {
		fakeUpdateServiceBrokerActor = &v7fakes.FakeActor{}
		fakeSharedActor = &commandfakes.FakeSharedActor{}
		input = NewBuffer()
		testUI = ui.NewTestUI(input, NewBuffer(), NewBuffer())
		fakeConfig = &commandfakes.FakeConfig{}
		cmd = &v7.UpdateServiceBrokerCommand{
			BaseCommand: v7.BaseCommand{
				Actor:       fakeUpdateServiceBrokerActor,
				SharedActor: fakeSharedActor,
				UI:          testUI,
				Config:      fakeConfig,
			},
		}
	})

	When("logged in", func() {
		const guid = "fake-service-broker-guid"

		BeforeEach(func() {
			fakeUpdateServiceBrokerActor.GetServiceBrokerByNameReturns(
				resources.ServiceBroker{GUID: guid},
				v7action.Warnings{},
				nil,
			)

			fakeUpdateServiceBrokerActor.GetCurrentUserReturns(configv3.User{Name: "user"}, nil)

			setPositionalFlags(cmd, serviceBrokerName, username, password, url)
		})

		It("succeeds", func() {
			fakeUpdateServiceBrokerActor.UpdateServiceBrokerReturns(v7action.Warnings{"update service broker warning"}, nil)

			err := cmd.Execute(nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeUpdateServiceBrokerActor.UpdateServiceBrokerCallCount()).To(Equal(1))
			serviceBrokerGUID, model := fakeUpdateServiceBrokerActor.UpdateServiceBrokerArgsForCall(0)
			Expect(serviceBrokerGUID).To(Equal(guid))
			Expect(model.Username).To(Equal(username))
			Expect(model.Password).To(Equal(password))
			Expect(model.URL).To(Equal(url))

			Expect(testUI.Err).To(Say("update service broker warning"))
		})

		When("the UpdateServiceBroker actor fails to get the broker name", func() {
			BeforeEach(func() {
				fakeUpdateServiceBrokerActor.GetServiceBrokerByNameReturns(
					resources.ServiceBroker{},
					v7action.Warnings{"some-warning"},
					actionerror.ServiceBrokerNotFoundError{
						Name: serviceBrokerName,
					},
				)
			})

			It("returns the error and displays all warnings", func() {
				err := cmd.Execute(nil)
				Expect(err).To(MatchError(actionerror.ServiceBrokerNotFoundError{Name: serviceBrokerName}))
				Expect(testUI.Err).To(Say("some-warning"))

				Expect(fakeUpdateServiceBrokerActor.GetServiceBrokerByNameCallCount()).To(Equal(1))
				Expect(fakeUpdateServiceBrokerActor.GetServiceBrokerByNameArgsForCall(0)).To(Equal(serviceBrokerName))
			})
		})

		When("the UpdateServiceBroker actor fails to update the broker", func() {
			It("returns the error and displays any warnings", func() {
				fakeUpdateServiceBrokerActor.UpdateServiceBrokerReturns(v7action.Warnings{"a-warning"}, errors.New("something went wrong"))

				err := cmd.Execute(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("something went wrong"))
				Expect(testUI.Err).To(Say("a-warning"))
			})
		})

		When("it fails to get the current user", func() {
			BeforeEach(func() {
				fakeUpdateServiceBrokerActor.GetCurrentUserReturns(configv3.User{}, errors.New("no user found"))
			})

			It("returns the error and displays all warnings", func() {
				err := cmd.Execute(nil)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("no user found"))
			})
		})

		When("password is provided as environment variable", func() {
			const (
				varName     = "CF_BROKER_PASSWORD"
				varPassword = "var-password"
			)

			BeforeEach(func() {
				setPositionalFlags(cmd, serviceBrokerName, username, url, "")
				os.Setenv(varName, varPassword)

				Expect(cmd.Execute(nil)).To(Succeed())
			})

			AfterEach(func() {
				os.Unsetenv(varName)
			})

			It("passes the data to the actor layer", func() {
				Expect(fakeUpdateServiceBrokerActor.UpdateServiceBrokerCallCount()).To(Equal(1))
				serviceBrokerGUID, model := fakeUpdateServiceBrokerActor.UpdateServiceBrokerArgsForCall(0)
				Expect(serviceBrokerGUID).To(Equal(guid))
				Expect(model.Username).To(Equal(username))
				Expect(model.Password).To(Equal(varPassword))
				Expect(model.URL).To(Equal(url))
			})
		})

		When("password is provided via prompt", func() {
			const promptPassword = "prompt-password"

			BeforeEach(func() {
				setPositionalFlags(cmd, serviceBrokerName, username, url, "")

				_, err := input.Write([]byte(fmt.Sprintf("%s\n", promptPassword)))
				Expect(err).NotTo(HaveOccurred())

				Expect(cmd.Execute(nil)).To(Succeed())
			})

			It("prompts the user for credentials", func() {
				Expect(testUI.Out).To(Say("Service Broker Password: "))
			})

			It("does not echo the credentials", func() {
				Expect(testUI.Out).NotTo(Say(promptPassword))
				Expect(testUI.Err).NotTo(Say(promptPassword))
			})

			It("passes the data to the actor layer", func() {
				Expect(fakeUpdateServiceBrokerActor.UpdateServiceBrokerCallCount()).To(Equal(1))
				serviceBrokerGUID, model := fakeUpdateServiceBrokerActor.UpdateServiceBrokerArgsForCall(0)
				Expect(serviceBrokerGUID).To(Equal(guid))
				Expect(model.Username).To(Equal(username))
				Expect(model.Password).To(Equal(promptPassword))
				Expect(model.URL).To(Equal(url))
			})
		})
	})

	When("not logged in", func() {
		BeforeEach(func() {
			fakeSharedActor.CheckTargetReturns(actionerror.NotLoggedInError{
				BinaryName: binaryName,
			})
		})

		It("returns an error", func() {
			err := cmd.Execute(nil)

			Expect(err).To(MatchError(actionerror.NotLoggedInError{BinaryName: binaryName}))

			Expect(fakeSharedActor.CheckTargetCallCount()).To(Equal(1))
			checkTargetedOrg, checkTargetedSpace := fakeSharedActor.CheckTargetArgsForCall(0)
			Expect(checkTargetedOrg).To(BeFalse())
			Expect(checkTargetedSpace).To(BeFalse())
		})
	})
})
