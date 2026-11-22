package main

import "time"

type PlatformOps interface {
	LockScreen()
	GetIdleDuration() time.Duration
	GetForegroundWindowTitle() string
	ShowNotification(title, message string)
}
