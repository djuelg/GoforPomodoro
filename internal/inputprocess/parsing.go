// This file is part of GoforPomodoro.
//
// GoforPomodoro is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// GoforPomodoro is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with GoforPomodoro.  If not, see <http://www.gnu.org/licenses/>.

package inputprocess

import (
	"GoforPomodoro/internal/domain"
	"GoforPomodoro/internal/utils"
	"regexp"
	"strconv"
	"strings"
)

const BasicPattern = `\/([1-9]\d*)(for([1-9]\d*)(rest([1-9]\d*))?)?` // `\/([1-9]\d*)`

const (
	MinutesGroup     = 1
	CardinalityGroup = 3
	RestGroup        = 5
)

var privacySettingsCommands = "/accept_all::/accept_essential"

func IsPrivacySettingsCommand(text string) bool {
	commands := strings.Split(privacySettingsCommands, "::")

	return utils.Contains(commands, text)
}

func CommandFrom(appSettings *domain.AppSettings, text string) string {
	command := strings.Split(text, " ")[0]

	botName := appSettings.BotName
	if !strings.HasPrefix(botName, "@") {
		botName = "@" + botName
	}

	if strings.HasSuffix(command, botName) {
		command = strings.Split(command, "@")[0]
	}

	return command
}

func ParametersFrom(text string) []string {
	return strings.Split(text, " ")[1:]
}

func ParsePatternToSession(r *regexp.Regexp, text string) utils.Optional[domain.SessionDefaultData] {
	if r == nil {
		r = regexp.MustCompile(BasicPattern)
	}
	matches := r.FindAllStringSubmatch(text, -1)

	var sessionDefaultData domain.SessionDefaultData

	match := false
	for _, v := range matches {
		match = true

		sessionDefaultData.SprintDurationSet = 1

		// Mandatory parameter for this command.
		pomDuration, err := strconv.Atoi(v[MinutesGroup])
		if err != nil {
			return utils.OptionalOfNil[domain.SessionDefaultData]()
		}
		sessionDefaultData.PomodoroDurationSet = domain.PomodoroDuration(pomDuration * 60) // time from minutes to seconds.

		// Other parameters are optional
		sprintDuration, err := strconv.Atoi(v[CardinalityGroup])
		if err == nil {
			sessionDefaultData.SprintDurationSet = domain.SprintDuration(sprintDuration)

			// Default 5 minutes of rest duration in case user did not specify.
			sessionDefaultData.RestDurationSet = 5 * 60
		}

		restDuration, err := strconv.Atoi(v[RestGroup])
		if err == nil {
			sessionDefaultData.RestDurationSet = domain.RestDuration(restDuration * 60)
		}

		break
	}

	if !match {
		return utils.OptionalOfNil[domain.SessionDefaultData]()
	}

	return utils.OptionalOf(sessionDefaultData)
}
