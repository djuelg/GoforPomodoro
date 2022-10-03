package sessionmanager

import (
	"GoforPomodoro/internal/domain"
	"errors"
	"log"
	"time"
)

type PomodoroEndKind int

const (
	PomodoroFinished PomodoroEndKind = iota
	PomodoroCanceled
)

func StartSession(
	userId domain.ChatID,
	currentSession *domain.Session,
	restBeginHandler func(id domain.ChatID, session *domain.Session),
	restFinishedHandler func(id domain.ChatID, session *domain.Session),
	endSessionHandler func(id domain.ChatID, session *domain.Session, endKind PomodoroEndKind),
	pauseSessionHandler func(id domain.ChatID, session *domain.Session),
) error {
	if currentSession.IsZero() {
		return errors.New("the session is effectively nil")
	}

	// How a sessionDefault is defined:
	// SprintDuration   int
	// PomodoroDuration int
	// RestDuration     int

	// We want
	// 1. Decrease Sprint duration by 1
	// 2. set a timer of PomodoroDuration minutes
	// 3. at its end, set a timer of PomodoroDuration seconds
	// 4. at its end, check if Sprint duration is >0. If so, go to 1, otherwise isPaused.
	currentSession.Data.IsPaused = false
	currentSession.Data.IsCancel = false

	currentSession.Data.SprintDuration -= 1

	currentSession.AssignTimestamps()

	go SpawnSessionTimer(
		userId,
		currentSession,
		restBeginHandler,
		restFinishedHandler,
		endSessionHandler,
		pauseSessionHandler,
	)
	return nil
}

func SpawnSessionTimer(
	userId domain.ChatID,
	currentSession *domain.Session,
	restBeginHandler func(id domain.ChatID, session *domain.Session),
	restFinishedHandler func(id domain.ChatID, session *domain.Session),
	endSessionHandler func(id domain.ChatID, session *domain.Session, endKind PomodoroEndKind),
	pauseSessionHandler func(id domain.ChatID, session *domain.Session),
) {
	sData := &currentSession.Data
mainLoop:
	for {
		select {
		case action, ok := <-currentSession.ReadingActionChannel():
			if ok {
				// The event was internal (rest started/finished)
				if action.RestStarted || action.RestFinished {
					if action.RestStarted {
						sData.IsRest = true
						sData.RestDuration = currentSession.RestDurationSet
						restBeginHandler(userId, currentSession)
					}
					if action.RestFinished {
						sData.IsRest = false
						sData.PomodoroDuration = currentSession.PomodoroDurationSet
						restFinishedHandler(userId, currentSession)
					}
					currentSession.AssignTimestamps()
					continue mainLoop
				}

				// The event was either external (paused/canceled) or internal (finished)
				if action.Paused || action.Canceled || action.Finished {
					if action.Paused {
						// Cache pomodoro and rest duration. We will use them again to assign new timestamps.
						sData.PomodoroDuration = currentSession.GetPomodoroDuration()
						sData.RestDuration = currentSession.GetRestDuration()

						sData.IsPaused = true
						pauseSessionHandler(userId, currentSession)
					} else if action.Canceled {
						sData.IsCancel = true
						endSessionHandler(userId, currentSession, PomodoroCanceled)
					} else if action.Finished {
						sData.IsFinished = true
						endSessionHandler(userId, currentSession, PomodoroFinished)
					}
					break mainLoop
				}
			} else {
				currentSession.ActionsChannel = nil
				log.Println("Session channel is closed. Aborting main loop...")
				break mainLoop
			}
		default:
			time.Sleep(1 * time.Second)

			isRest := currentSession.Data.IsRest

			if !isRest && time.Now().Local().After(*currentSession.EndNextSprintTimestamp) {
				currentSession.Data.SprintDuration -= 1

				if currentSession.Data.SprintDuration < 0 {
					currentSession.WritingActionChannel() <- domain.DispatchAction{Finished: true}
					continue mainLoop
				}

				// if SprintDuration still >= 0, we have rest now
				currentSession.WritingActionChannel() <- domain.DispatchAction{RestStarted: true}
				continue mainLoop
			} else if isRest && time.Now().Local().After(*currentSession.EndNextRestTimestamp) {

				currentSession.WritingActionChannel() <- domain.DispatchAction{RestFinished: true}
				continue mainLoop
			}
		}
	}
	defer func() {
		close(currentSession.ActionsChannel)
		currentSession.ActionsChannel = nil
	}()
}

func PauseSession(currentSession *domain.Session) error {
	if currentSession.Data.IsPaused {
		return errors.New("sessionDefault already paused")
	}

	currentSession.WritingActionChannel() <- domain.DispatchAction{Paused: true}
	return nil
}

func CancelSession(currentSession *domain.Session) error {
	if currentSession.IsCanceled() {
		return errors.New("sessionDefault already canceled")
	}

	currentSession.WritingActionChannel() <- domain.DispatchAction{Canceled: true}
	return nil
}

func ResumeSession(
	userId domain.ChatID,
	currentSession *domain.Session,
	restBeginHandler func(id domain.ChatID, session *domain.Session),
	restFinishedHandler func(id domain.ChatID, session *domain.Session),
	endSessionHandler func(id domain.ChatID, session *domain.Session, endKind PomodoroEndKind),
	pauseSessionHandler func(id domain.ChatID, session *domain.Session),
) error {
	if currentSession.IsZero() {
		return errors.New("the session is effectively nil")
	}
	if !currentSession.IsStopped() {
		return errors.New("session already running")
	}
	if currentSession.IsCanceled() {
		return errors.New("session was canceled")
	}

	currentSession.Data.IsPaused = false

	currentSession.AssignTimestamps()

	go SpawnSessionTimer(
		userId,
		currentSession,
		restBeginHandler,
		restFinishedHandler,
		endSessionHandler,
		pauseSessionHandler,
	)
	return nil
}
