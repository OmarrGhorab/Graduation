package jobs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/OmarrGhorab/courses-attendance-service/internal/domain/events"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/clock"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/notificationevents"
	"github.com/OmarrGhorab/courses-attendance-service/internal/infrastructure/persistence/postgres"
)

// LessonRemindersJob scans for upcoming lessons and sends custom reminders
type LessonRemindersJob struct {
	lessonRepo      *postgres.LessonRepository
	eventDispatcher *notificationevents.EventDispatcher
	clock           clock.Clock
}

func NewLessonRemindersJob(
	lessonRepo *postgres.LessonRepository,
	eventDispatcher *notificationevents.EventDispatcher,
	clk clock.Clock,
) *LessonRemindersJob {
	return &LessonRemindersJob{
		lessonRepo:      lessonRepo,
		eventDispatcher: eventDispatcher,
		clock:           clk,
	}
}

// Run executes the reminder logic for the current time
func (j *LessonRemindersJob) Run(ctx context.Context) error {
	now := j.clock.Now()

	// Fetch lessons scheduled for the next 24 hours to catch any custom reminders
	start := now
	end := now.Add(24 * time.Hour)

	lessons, err := j.lessonRepo.GetLessonsWithCourseIntervals(ctx, start, end)
	if err != nil {
		log.Printf("[LessonRemindersJob] Error fetching lessons: %v", err)
		return err
	}

	for _, l := range lessons {
		j.processLessonReminders(ctx, l, now)
	}

	return nil
}

func (j *LessonRemindersJob) processLessonReminders(ctx context.Context, item postgres.LessonWithIntervals, now time.Time) {
	l := item.Lesson
	intervals := strings.Split(item.ReminderIntervals, ",")
	sentList := strings.Split(l.RemindersSent, ",")

	updatedSent := sentList
	needsUpdate := false

	for _, intervalStr := range intervals {
		intervalStr = strings.TrimSpace(intervalStr)
		if intervalStr == "" {
			continue
		}

		var minutes int
		_, err := fmt.Sscanf(intervalStr, "%d", &minutes)
		if err != nil {
			continue
		}

		// Check if already sent
		isSent := false
		for _, s := range sentList {
			if s == intervalStr {
				isSent = true
				break
			}
		}
		if isSent {
			continue
		}

		// Check if it's time to send (within a 2-minute window of the target time)
		targetTime := l.ScheduledAt.Add(-time.Duration(minutes) * time.Minute)
		if now.After(targetTime.Add(-1*time.Minute)) && now.Before(targetTime.Add(1*time.Minute)) {
			// Send reminder
			j.eventDispatcher.EmitLessonReminder(ctx, events.LessonReminderPayload{
				LessonID:      l.ID,
				CourseID:      l.CourseID,
				LessonTitle:   l.Title,
				MinutesBefore: minutes,
				ScheduledAt:   l.ScheduledAt,
			})

			updatedSent = append(updatedSent, intervalStr)
			needsUpdate = true
			log.Printf("[LessonRemindersJob] Sent %dm reminder for lesson %s", minutes, l.ID)
		}
	}

	if needsUpdate {
		l.RemindersSent = strings.Join(updatedSent, ",")
		if err := j.lessonRepo.Update(ctx, &l); err != nil {
			log.Printf("[LessonRemindersJob] Error updating lesson %s: %v", l.ID, err)
		}
	}
}

// StartScheduler starts a ticker to run the job periodically
func (j *LessonRemindersJob) StartScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[LessonRemindersJob] Starting scheduler with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Println("[LessonRemindersJob] Scheduler stopped")
			return
		case <-ticker.C:
			if err := j.Run(ctx); err != nil {
				log.Printf("[LessonRemindersJob] Job failed: %v", err)
			}
		}
	}
}
