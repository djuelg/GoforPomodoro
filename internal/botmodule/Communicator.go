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
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"strings"
	"time"
)

var simpleHourglassKeyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("⌛", "⌛"),
	),
)

type Communicator struct {
	appState     *domain.AppState
	appVariables *domain.AppVariables
	ChatID       domain.ChatID
	Bot          *tgbotapi.BotAPI
	Subscribers  []domain.ChatID
	IsGroup      bool
}

func GetCommunicator(appState *domain.AppState, appVariables *domain.AppVariables, chatId domain.ChatID, bot *tgbotapi.BotAPI) *Communicator {
	communicator := new(Communicator)

	communicator.appState = appState
	communicator.appVariables = appVariables
	communicator.ChatID = chatId
	communicator.Bot = bot
	communicator.Subscribers = data.GetSubscribers(appState, chatId)
	communicator.IsGroup = data.IsGroup(appState, chatId)

	return communicator
}

func (c *Communicator) subscribersAsString() string {
	bot := c.Bot

	var sb strings.Builder

	errors := 0
	for _, id := range c.Subscribers {
		subscriberChat, err := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: int64(id)}})
		if err != nil {
			errors += 1
			continue
		}
		sb.WriteString("@")
		sb.WriteString(subscriberChat.UserName)
		sb.WriteString(" ")
	}

	return sb.String()
}

func (c *Communicator) toNotify(message string) string {
	// Update subscribers in case they changed
	c.Subscribers = data.GetSubscribers(c.appState, c.ChatID)

	if !c.IsGroup || len(c.Subscribers) == 0 {
		// This function is identity function if we're not in a group or there are no subscribers.
		return message
	}

	return message + "\n\n———\n" + c.subscribersAsString()
}

func (c *Communicator) Subscribe(err error, update tgbotapi.Update, username string) {
	if err != nil {
		switch err.Error() {
		case domain.AlreadySubscribed{}.Error():
			c.ReplyWith("Du hast diese Gruppe schon subscribed" +
				"Mit dem Befehl /leave kannst du die Subscription beenden.")
		case domain.SubscriptionError{}.Error():
			c.ReplyWith("There has been an error with this operation (subscription).")
		}
	} else {
		c.ReplyWith(fmt.Sprintf("Erledigt! Du (@%s) wirst in den Sprint-Nachrichten getaggt.", username))
	}
}

func (c *Communicator) Unsubscribe(err error) {
	if err != nil {
		switch err.Error() {
		case domain.AlreadyUnsubscribed{}.Error():
			c.ReplyWith("Du bist gerade nicht subscribed in dieser Gruppe.")
		case domain.SubscriptionError{}.Error():
			c.ReplyWith("There has been an error with this operation (subscription).")
		}
	} else {
		c.ReplyWith("Erledigt! Du wirst nicht mehr in den Sprint-Nachrichten getaggt.")
	}
}

func (c *Communicator) ReplyWith(text string) {
	bot := c.Bot
	chatId := int64(c.ChatID)

	msg := tgbotapi.NewMessage(chatId, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}
}

func (c *Communicator) ReplyWithSticker(stickerID string) {
	bot := c.Bot
	chatId := int64(c.ChatID)

	msg := tgbotapi.NewSticker(chatId, tgbotapi.FileID(stickerID))
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}
}

func (c *Communicator) ReplyWithParseMode(text string, parseMode string, disablePreview bool) {
	bot := c.Bot
	chatId := int64(c.ChatID)

	msg := tgbotapi.NewMessage(chatId, text)
	msg.ParseMode = parseMode
	msg.DisableWebPagePreview = disablePreview
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}
}

func (c *Communicator) ReplyAndNotify(text string) {
	c.ReplyWith(c.toNotify(text))
}

func (c *Communicator) ReplyWithAndHourglass(text string) {
	msg := tgbotapi.NewMessage(int64(c.ChatID), text)
	msg.ReplyMarkup = simpleHourglassKeyboard
	_, err := c.Bot.Send(msg)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
	}
}

func (c *Communicator) ReplyWithAndHourglassAndNotify(text string) {
	c.ReplyWithAndHourglass(c.toNotify(text))
}

func (c *Communicator) SessionStarted(session *domain.Session, err error) {
	if err == nil {
		sessionTime := session.CalculateSessionTimeInSeconds()
		var replyStr string

		if session.IsSprintDurationUnspecified() {
			replyStr = "Diese Session wird so lange dauern, wie du fokussiert bleiben möchtest."
		} else {
			replyStr = fmt.Sprintf("Diese Session wird %s dauern\n\nSession gestartet!", utils.NiceTimeFormatting64(sessionTime))
		}
		c.ReplyWithAndHourglassAndNotify(replyStr)
	} else {
		c.ReplyWith("Session wurde nicht gestartet.\nBitte erstelle eine Session oder benutze /default für das klassische 4x25m+5m.")
	}
}

/*
func (c *Communicator) SessionFinished() {

}

func (c *Communicator) SessionResumed() {

}

func (c *Communicator) SessionPaused() {

}*/

func (c *Communicator) SessionFinishedHandler(id domain.ChatID, session *domain.Session, endKind sessionmanager.PomodoroEndKind) {
	switch endKind {
	case sessionmanager.PomodoroFinished:
		c.ReplyWithSticker("CAACAgIAAxkBAAEss6tmm63I_lXy3RWKz85flZjDNmMyxgAC9zQAAoPYKUg5dyrC1UyEyDUE")
		c.ReplyAndNotify("Pomodoro erledigt. Die Session ist abgeschlossen, Glückwunsch!")
	case sessionmanager.PomodoroCanceled:
		c.ReplyAndNotify("Session abgebrochen.")
	}
}

func (c *Communicator) SessionPausedHandler(id domain.ChatID, session *domain.Session) {
	c.ReplyAndNotify("Deine Session wurde pausiert.")
}

func (c *Communicator) RestFinishedHandler(id domain.ChatID, session *domain.Session) {
	text := fmt.Sprintf(
		"Pomodoro %s gestartet.",
		utils.NiceTimeFormatting(session.GetPomodoroDurationSet().Seconds()),
	)
	c.ReplyWithAndHourglassAndNotify(text)
}

func (c *Communicator) RestBeginHandler(id domain.ChatID, session *domain.Session) {
	text := fmt.Sprintf(
		"Pomodoro erledigt! Du kannst jetzt für %s Pause machen.",
		utils.NiceTimeFormatting(session.GetRestDurationSet().Seconds()),
	)

	c.ReplyAndNotify(text)
}

func (c *Communicator) SessionAlreadyRunning() {
	c.ReplyWith("Es läuft bereits eine Session.")
}

func (c *Communicator) SessionResumed(err error, session *domain.Session) {
	if err != nil {
		if session.IsZero() {
			c.ReplyWith("Session wurde nicht gestartet.")
		} else if session.IsCanceled() {
			c.ReplyWith("Die letzte Session wurde abgebrochen")
		} else if !session.IsStopped() {
			c.ReplyWith("Es läuft bereits eine Session.")
		} else {
			c.ReplyWith("Server error.")
		}
		return
	}

	c.ReplyWithAndHourglassAndNotify("Session fortgesetzt!")
}

func (c *Communicator) OnlyGroupsCommand() {
	c.ReplyWith("Dieser Befehl funktioniert nur in Gruppen, sorry.")
}

func (c *Communicator) NewSession(session domain.SessionDefaultData) {
	c.ReplyWithSticker("CAACAgIAAxkBAAEstLZmm_IKhOhXcYWUgdpy6xs6SfVNywAC_1AAAr9S4UiHr0g8ntWq3jUE")
	c.ReplyWith(fmt.Sprintf("Viel Erfolg bei der neuen Session!\n\n%s", session.String()))
}

func (c *Communicator) Info() {
	c.ReplyWith("Hi, schön dich kennenzulernen! Ich bin ein Bot der versucht dir dabei zu helfen effizienter zu arbeiten.")
	c.ReplyWithSticker("CAACAgIAAxkBAAEstLZmm_IKhOhXcYWUgdpy6xs6SfVNywAC_1AAAr9S4UiHr0g8ntWq3jUE")
	c.ReplyWith("Ich unterstütze mit Hilfe der Pomodoro Technik, einer Methode zur Steigerung der Produktivität durch Aufteilung der Arbeitszeit in Intervalle mit kurzen Pausen dazwischen. Außerdem helfe ich mit Timeboxing, einer Strategie, bei der für eine Aufgabe im voraus ein festes, begrenztes Zeitfenster eingeplant wird. Dies soll helfen sich nicht zu lange in Details zu verlieren.")
}

func (c *Communicator) DataCleaned() {
	c.ReplyWith("Deine Daten wurden gelöscht.")
}

func (c *Communicator) Help() {
	c.ReplyWith("Erstelle eine Session mit frei wählbaren Zeiten (z.B.)\n" +
		"/25for4rest5 ➡️ 4 Pomodoros, je 25 Minuten + 5 Minuten Pause.\n" +
		"Dies ist die Standardeinstellung, die auch mit /default gestartet werden kann.\n" +
		"/30for4 ➡️ 4 Pomodoros, je 30 Minuten (mit Default von +5m als Pause).\n" +
		"/25 ➡️ 1 Pomodoro, 25 Minuten (einzelner Sprint)\n\n" +
		"Session Verwaltung:\n" +
		"(/s) /start_sprint um einen Sprint zu starten (wenn /autorun ausgeschaltet ist)\n" +
		"(/p) /pause um eine Session zu pausieren\n" +
		"(/c) /cancel um eine Session abzubrechen\n" +
		"/resume um eine pausierte Session fortzusetzen\n" +
		"(/se) /session um den aktuellen Status der Session zu sehen\n\n" +
		"Weitere Commands \n" +
		"(/tb) /timebox HH:MM Beschreibung ➡️ Erstellt eine Timebox bis zur angegebenen Uhrzeit \n" +
		"/reset um deine Daten zu löschen\n" +
		"/info um mehr über mich zu erfahren")
}

func (c *Communicator) SessionPaused(err error, session domain.Session) {
	if err != nil {
		if !session.IsStopped() {
			c.ReplyWith("Es gibt keine laufende Session.")
		} else {
			c.ReplyWith("Server error.")
		}
	}
}

func (c *Communicator) SessionCanceled(err error, session domain.Session) {
	if err != nil {
		if session.IsStopped() {
			c.ReplyWith("Es gibt keine laufende Session.")
		} else {
			c.ReplyWith("Server error.")
		}
	}
}

func (c *Communicator) SessionState(session domain.Session) {
	var stateStr = session.State()

	var replyMsgText string
	if session.IsCanceled() {
		replyMsgText = fmt.Sprintf("Aktueller Zustand deiner Session: %s.", stateStr)
	} else {
		replyMsgText = session.String()
	}
	c.ReplyWith(replyMsgText)
}

func (c *Communicator) CommandError() {
	c.ReplyWith("Command error.")
}

func (c *Communicator) Hourglass() {
	c.ReplyWithAndHourglass("Hier ist eine Sanduhr")
}

func (c *Communicator) ShowPrivacyPolicy() {
	c.ReplyWithParseMode(c.appVariables.PrivacyPolicy1, "html", true)
}

func (c *Communicator) PrivacySettingsUpdated() {
	c.ReplyWith("Deine Datenschutzeinstellungen wurden aktualisiert.")
}

func (c *Communicator) ShowLicenseNotice() {
	c.ReplyWithParseMode(c.appVariables.OpenSource1, "html", true)
}

func (c *Communicator) ErrorSessionTooLong() {
	tooLongMessages := [...]string{
		"Die Session ist zu lang.",
		"Mit dieser Länge der Session kann ich nicht umgehen.",
	}
	rand.Seed(time.Now().UnixNano())
	c.ReplyWith(tooLongMessages[rand.Intn(len(tooLongMessages))])
}
