variable "base_path" {
  description = "API route prefix stripped by the Lambda adapter"
  type        = string
  default     = "/perthfuel"
}

variable "allowed_origin" {
  description = "Origin allowed by API Gateway CORS"
  type        = string
  default     = "https://gofuel.seanboaden.dev"
}
