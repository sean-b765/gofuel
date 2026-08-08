output "github-cicd-role" {
  description = "OpenID Connect role to assume in github actions"
  value       = aws_iam_role.github_actions_role.arn
}
