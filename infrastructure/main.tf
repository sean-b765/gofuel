resource "aws_dynamodb_table" "public" {
  name           = "Stations"
  billing_mode   = "PROVISIONED"
  read_capacity  = 2
  write_capacity = 5

  tags = {
    Project = "gofuel"
  }

  hash_key  = "RegionGeohash"
  range_key = "TownGeohash"

  attribute {
    name = "RegionGeohash"
    type = "S"
  }

  attribute {
    name = "TownGeohash"
    type = "S"
  }

  attribute {
    name = "SubRegionGeohash"
    type = "S"
  }

  global_secondary_index {
    name               = "GSI_SubRegion"
    hash_key           = "SubRegionGeohash"
    range_key          = "TownGeohash"
    projection_type    = "INCLUDE"
    non_key_attributes = ["StationId", "Title", "Brand", "Address", "Ulp91", "Ulp95", "Ulp98", "Diesel"]
    read_capacity      = 2
    write_capacity     = 5
  }
}

resource "aws_api_gateway_rest_api" "this" {
  name = "gofuel"

  endpoint_configuration {
    types = ["EDGE"]
  }
}

resource "aws_api_gateway_resource" "root" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  path_part   = ""
  parent_id   = ""
}

# __generated__ by Terraform from "njwn9hkiph/tv01xg"
resource "aws_api_gateway_resource" "proxy" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  path_part   = "{proxy+}"
  parent_id   = aws_api_gateway_resource.root.id
}

# __generated__ by Terraform from "njwn9hkiph/tv01xg/ANY"
resource "aws_api_gateway_method" "any" {
  api_key_required     = false
  authorization        = "NONE"
  authorization_scopes = []
  authorizer_id        = null
  http_method          = "ANY"
  operation_name       = null
  request_models       = {}
  request_parameters = {
    "method.request.path.proxy" = true
  }
  request_validator_id = null
  resource_id          = aws_api_gateway_resource.proxy.id
  rest_api_id          = aws_api_gateway_rest_api.this.id
}

# __generated__ by Terraform from "njwn9hkiph/c6gk9famag"

# __generated__ by Terraform from "njwn9hkiph/tv01xg/ANY"
resource "aws_api_gateway_integration" "any_integration" {
  http_method             = "ANY"
  integration_http_method = "POST"
  passthrough_behavior    = "WHEN_NO_MATCH"
  request_parameters      = {}
  request_templates       = {}
  resource_id             = aws_api_gateway_resource.proxy.id
  rest_api_id             = aws_api_gateway_rest_api.this.id
  timeout_milliseconds    = 15000
  type                    = "AWS_PROXY"
  uri                     = "arn:aws:apigateway:ap-southeast-2:lambda:path/2015-03-31/functions/arn:aws:lambda:ap-southeast-2:476720619618:function:gofuel/invocations"
}

# __generated__ by Terraform from "njwn9hkiph/prod"
resource "aws_api_gateway_stage" "prod" {
  stage_name           = "prod"
  description          = "Production"
  deployment_id        = "udi98p"
  rest_api_id          = aws_api_gateway_rest_api.this.id
  tags                 = {}
  tags_all             = {}
  variables            = {}
  xray_tracing_enabled = false
}

resource "aws_lambda_function" "this" {
  architectures = ["x86_64"]
  description   = "GoFuel API"
  function_name = "gofuel"
  image_uri     = "476720619618.dkr.ecr.ap-southeast-2.amazonaws.com/gofuel:055c4322d52ef7bac7bc04b0b17546487d286982"
  memory_size   = 128
  package_type  = "Image"
  role          = "arn:aws:iam::476720619618:role/service-role/gofuel-role-jw1dcxrc"
  runtime       = null
  tags          = {}
  timeout       = 3
  ephemeral_storage {
    size = 512
  }
  logging_config {
    application_log_level = null
    log_format            = "Text"
    log_group             = "/aws/lambda/gofuel"
    system_log_level      = null
  }
  tracing_config {
    mode = "Active"
  }
}
