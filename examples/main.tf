terraform {
  required_providers {
    habitica = {
      source = "registry.terraform.io/inannamalick/habitica"
    }
  }
}

provider "habitica" {
  # These can also be set via environment variables:
  # HABITICA_USER_ID, HABITICA_API_TOKEN, HABITICA_CLIENT_AUTHOR_ID
  user_id          = var.habitica_user_id
  api_token        = var.habitica_api_token
  client_author_id = var.habitica_client_author_id
  client_app_name  = "TerraformHabitica"
  rate_limit_buffer = 5
}

variable "habitica_user_id" {
  type        = string
  description = "Your Habitica user ID (UUID)"
}

variable "habitica_api_token" {
  type        = string
  description = "Your Habitica API token"
  sensitive   = true
}

variable "habitica_client_author_id" {
  type        = string
  description = "Your Habitica user ID for the x-client header"
}

# Tags for organizing tasks
resource "habitica_tag" "health" {
  name = "Health"
}

resource "habitica_tag" "work" {
  name = "Work"
}

# Habits - positive/negative scoring
resource "habitica_habit" "water" {
  text     = "Drink water"
  notes    = "Stay hydrated throughout the day"
  priority = 1    # Easy difficulty
  up       = true
  down     = false
  tags     = [habitica_tag.health.id]
}

resource "habitica_habit" "posture" {
  text     = "Good posture"
  notes    = "Maintain good posture at desk"
  priority = 1
  up       = true
  down     = true  # Can also score negatively for slouching
  tags     = [habitica_tag.health.id, habitica_tag.work.id]
}

# Dailies - recurring scheduled tasks
resource "habitica_daily" "morning_workout" {
  text       = "Morning workout"
  notes      = "30 minutes of exercise"
  priority   = 1.5  # Medium difficulty
  frequency  = "weekly"
  every_x    = 1
  start_date = "2025-01-01"

  repeat {
    monday    = true
    tuesday   = true
    wednesday = true
    thursday  = true
    friday    = true
    saturday  = false
    sunday    = false
  }

  tags = [habitica_tag.health.id]
}

resource "habitica_daily" "monthly_review" {
  text          = "Monthly review"
  notes         = "Review goals and progress"
  priority      = 2  # Hard difficulty
  frequency     = "monthly"
  every_x       = 1
  days_of_month = [1]  # First day of each month

  tags = [habitica_tag.work.id]
}

# Webhooks - event notifications
resource "habitica_webhook" "task_notifications" {
  url     = "https://example.com/habitica-webhook"
  label   = "Task Activity"
  type    = "taskActivity"
  enabled = true

  options {
    created = false
    updated = false
    deleted = false
    scored  = true  # Only trigger on task scoring
  }
}

# Outputs
output "health_tag_id" {
  value = habitica_tag.health.id
}

output "work_tag_id" {
  value = habitica_tag.work.id
}
