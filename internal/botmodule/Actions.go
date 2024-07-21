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

package botmodule

import (
	"GoforPomodoro/internal/data"
	"GoforPomodoro/internal/domain"
	"GoforPomodoro/internal/sessionmanager"
	"GoforPomodoro/internal/utils"
	"time"
	"fmt"
)

func ActionRestoreSprint(
	chatId domain.ChatID,
	appState *domain.AppState,
	session *domain.Session,
	communicator *Communicator,
) {
	go sessionmanager.SpawnSessionTimer(
		appState,
		chatId,
		session,
		communicator.RestBeginHandler,
		communicator.RestFinishedHandler,
		communicator.SessionFinishedHandler,
		communicator.SessionPausedHandler,
	)
}

func ActionCancelSprint(
	senderId domain.ChatID,
	chatId domain.ChatID,
	appState *domain.AppState,
	communicator *Communicator,
) {
	session := data.GetUserSessionRunning(appState, chatId, senderId)

	var err error
	if !session.IsPaused() {
		err = sessionmanager.CancelSession(session)
	} else {
		session.Cancel()
		communicator.SessionFinishedHandler(chatId, session, sessionmanager.PomodoroCanceled)
	}

	communicator.SessionCanceled(err, *session)
}

func ActionResumeSprint(
	senderId domain.ChatID,
	chatId domain.ChatID,
	appState *domain.AppState,
	communicator *Communicator,
) {
	session := data.GetUserSessionRunning(appState, chatId, senderId)
	communicator.SessionResumed(
		sessionmanager.ResumeSession(
			appState,
			chatId,
			session,
			communicator.RestBeginHandler,
			communicator.RestFinishedHandler,
			communicator.SessionFinishedHandler,
			communicator.SessionPausedHandler,
		),
		session,
	)
}

func ActionStartSprint(
	senderId domain.ChatID,
	chatId domain.ChatID,
	appState *domain.AppState,
	communicator *Communicator,
) {

	// log.Printf("[NO-DB TEST] ActionStartSprint!!\n")
	session := data.GetUserSessionRunning(appState, chatId, senderId)

	// log.Printf("[NO-DB TEST] data.GetUserSessionRunning succeded\n")
	if !session.IsStopped() {
		communicator.SessionAlreadyRunning()
		// log.Printf("[NO-DB TEST] session already running: stopping\n")
		return
	}
	session = data.GetNewUserSessionRunning(appState, chatId, senderId)

	// log.Printf("[NO-DB TEST] new session running: %v\n", session)
	communicator.SessionStarted(
		session,
		sessionmanager.StartSession(
			appState,
			chatId,
			session,
			communicator.RestBeginHandler,
			communicator.RestFinishedHandler,
			communicator.SessionFinishedHandler,
			communicator.SessionPausedHandler,
		),
	)
}

func ActionTimebox(
	senderId domain.ChatID,
	chatId domain.ChatID,
	appState *domain.AppState,
	communicator *Communicator,
	timeStr string,
	taskDescription string,
) {

	// Identify German time zone (CET or CEST)
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		fmt.Println("Error loading location:", err)
		return
	}

	// Get the current date and time in the specified location
	now := time.Now().In(loc)

	// Parse the user's time input in the specified location
	userTime, err := time.ParseInLocation("15:04", timeStr, loc)
	if err != nil {
		fmt.Println("Error parsing time:", err)
		return
	}

	// Combine current date with the parsed time
	userDateTime := time.Date(now.Year(), now.Month(), now.Day(), userTime.Hour(), userTime.Minute(), 0, 0, loc)

	// If the parsed time is earlier today, add one day
	if userDateTime.Before(now) {
		userDateTime = userDateTime.Add(24 * time.Hour)
	}

	// Calculate the duration until the specified time
	duration := userDateTime.Sub(now)
	msg := fmt.Sprintf("Timebox erstellt\n→ Erinnerung in %s", utils.NiceTimeFormatting(int(duration.Seconds())))
	communicator.ReplyWith(msg)

	// Schedule the reminder
	go func() {
		if duration > 0 {
			time.Sleep(duration)
			msg := fmt.Sprintf("Timebox abgelaufen ✨\n→ %s", taskDescription)
			communicator.ReplyWith(msg)
		}
	}()
}
