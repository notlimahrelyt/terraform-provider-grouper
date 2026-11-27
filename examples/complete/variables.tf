variable "parent_stem" {
  description = "Existing folder the application folder is created under, e.g. \"example:apps\"."
  type        = string
}

variable "app_id" {
  description = "Short system name for the application. Becomes the last component of the folder name."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9_-]*$", var.app_id))
    error_message = "app_id must be lowercase letters, digits, hyphens, or underscores."
  }
}

variable "app_name" {
  description = "Human-readable application name, used in display names and descriptions."
  type        = string
}

variable "admins" {
  description = "Subject ids or login identifiers to add to the admins group."
  type        = set(string)
  default     = []
}

variable "users" {
  description = "Subject ids or login identifiers to add to the users group."
  type        = set(string)
  default     = []
}
