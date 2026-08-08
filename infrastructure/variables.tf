variable "base_path" {
  description = "API route prefix stripped by the Lambda adapter"
  type        = string
  default     = "/gofuel"
}
variable "gh_repo" {
  description = "GitHub repo, like 'org_name/repo_name'"
  type        = string
  default     = "sean-b765/gofuel"
}
variable "allowed_origin" {
  description = "Origin allowed by API Gateway CORS"
  type        = string
  default     = "https://gofuel.seanboaden.dev"
}
