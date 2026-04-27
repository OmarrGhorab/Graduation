ALTER TABLE courses ADD COLUMN reminder_intervals VARCHAR(255) DEFAULT '15,10,5';
ALTER TABLE lessons ADD COLUMN reminders_sent TEXT DEFAULT '';
