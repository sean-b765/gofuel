# __generated__ by Terraform
# Please review these resources and move them into your main configuration files.

# __generated__ by Terraform from "njwn9hkiph/tv01xg/OPTIONS"
resource "aws_api_gateway_method" "options" {
  api_key_required     = false
  authorization        = "NONE"
  authorization_scopes = []
  authorizer_id        = null
  http_method          = "OPTIONS"
  operation_name       = null
  request_models       = {}
  request_parameters   = {}
  request_validator_id = null
  resource_id          = "tv01xg"
  rest_api_id          = "njwn9hkiph"
}

# 1. The MOCK Integration
resource "aws_api_gateway_integration" "options_integration" {
  rest_api_id = aws_api_gateway_method.options.rest_api_id
  resource_id = aws_api_gateway_method.options.resource_id
  http_method = aws_api_gateway_method.options.http_method
  type        = "MOCK"

  # Pass a basic 200 status code template for the mock integration
  request_templates = {
    "application/json" = jsonencode({ statusCode = 200 })
  }
}

resource "aws_api_gateway_method_response" "options_200" {
  rest_api_id = aws_api_gateway_method.options.rest_api_id
  resource_id = aws_api_gateway_method.options.resource_id
  http_method = aws_api_gateway_method.options.http_method
  status_code = "200"

  response_models = {
    "application/json" = "Empty"
  }

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Origin"  = true
  }
}

resource "aws_api_gateway_integration_response" "options_integration_response" {
  rest_api_id = aws_api_gateway_method.options.rest_api_id
  resource_id = aws_api_gateway_method.options.resource_id
  http_method = aws_api_gateway_method.options.http_method
  status_code = aws_api_gateway_method_response.options_200.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
    "method.response.header.Access-Control-Allow-Methods" = "'DELETE,GET,HEAD,OPTIONS,PATCH,POST,PUT'"
    "method.response.header.Access-Control-Allow-Origin"  = "'*'"
  }

  depends_on = [
    aws_api_gateway_integration.options_integration
  ]
}
