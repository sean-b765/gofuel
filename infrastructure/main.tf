resource "aws_dynamodb_table" "public" {
  name         = "Stations"
  billing_mode = "PAY_PER_REQUEST"

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
    non_key_attributes = ["StationId", "Title", "Brand", "Address", "Latitude", "Longitude", "Date", "Ulp91", "Ulp95", "Ulp98", "Diesel"]
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  lifecycle { prevent_destroy = true }
}

resource "aws_dynamodb_table" "auth" {
  name         = "Providers-Auth"
  billing_mode = "PAY_PER_REQUEST"

  tags = {
    Project = "gofuel"
  }

  hash_key = "provider"

  attribute {
    name = "provider"
    type = "S"
  }

  ttl {
    attribute_name = "ttl"
    enabled        = true
  }

  lifecycle { prevent_destroy = true }
}

# SSM Parameter Store: provider API keys

locals {
  secret_params = toset([
    "maps_key",
    "nsw_tas_api_key",
    "nsw_tas_api_secret",
    "sa_api_key",
    "qld_api_key",
  ])
}

resource "aws_ssm_parameter" "secrets" {
  for_each = local.secret_params

  name  = "/gofuel/${each.key}"
  type  = "String"
  value = "change-me"

  lifecycle {
    ignore_changes = [value]
  }

  tags = {
    Project = "gofuel"
  }
}

resource "aws_api_gateway_rest_api" "this" {
  name = "gofuel"

  endpoint_configuration {
    types = ["REGIONAL"]
  }

  tags = {
    Project = "gofuel"
  }
}

resource "aws_api_gateway_resource" "proxy" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  path_part   = "{proxy+}"
  parent_id   = aws_api_gateway_rest_api.this.root_resource_id
}

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

resource "aws_api_gateway_integration" "any_integration" {
  http_method             = aws_api_gateway_method.any.http_method
  integration_http_method = "POST"
  passthrough_behavior    = "WHEN_NO_MATCH"
  request_parameters      = {}
  request_templates       = {}
  resource_id             = aws_api_gateway_resource.proxy.id
  rest_api_id             = aws_api_gateway_rest_api.this.id
  timeout_milliseconds    = 15000
  type                    = "AWS_PROXY"
  uri                     = "arn:aws:apigateway:${data.aws_region.current.name}:lambda:path/2015-03-31/functions/arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:gofuel/invocations"
}

# API Gateway: CORS preflight

resource "aws_api_gateway_method" "options" {
  rest_api_id   = aws_api_gateway_rest_api.this.id
  resource_id   = aws_api_gateway_resource.proxy.id
  http_method   = "OPTIONS"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "options" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.proxy.id
  http_method = aws_api_gateway_method.options.http_method

  type                 = "MOCK"
  passthrough_behavior = "WHEN_NO_MATCH"
  request_templates = {
    "application/json" = "{\"statusCode\": 200}"
  }
}

resource "aws_api_gateway_method_response" "options" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.proxy.id
  http_method = aws_api_gateway_method.options.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = true
    "method.response.header.Access-Control-Allow-Methods" = true
    "method.response.header.Access-Control-Allow-Headers" = true
  }

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration_response" "options" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.proxy.id
  http_method = aws_api_gateway_method.options.http_method
  status_code = aws_api_gateway_method_response.options.status_code

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin"  = "'${var.allowed_origin}'"
    "method.response.header.Access-Control-Allow-Methods" = "'GET,POST,PUT,DELETE,PATCH,OPTIONS'"
    "method.response.header.Access-Control-Allow-Headers" = "'Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token'"
  }

  depends_on = [aws_api_gateway_integration.options]
}

# Allow the Lambda proxy to pass through the CORS header on real responses.
resource "aws_api_gateway_method_response" "any_200" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  resource_id = aws_api_gateway_resource.proxy.id
  http_method = aws_api_gateway_method.any.http_method
  status_code = "200"

  response_parameters = {
    "method.response.header.Access-Control-Allow-Origin" = true
  }
}

resource "aws_lambda_permission" "apigw" {
  statement_id  = "AllowExecutionFromAPIGateway"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.this.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.this.execution_arn}/*/*"
}

resource "aws_api_gateway_deployment" "this" {
  rest_api_id = aws_api_gateway_rest_api.this.id

  triggers = {
    redeployment = sha1(jsonencode([
      aws_api_gateway_rest_api.this.body,
      aws_api_gateway_method.options.id,
      aws_api_gateway_integration.options.id,
      aws_api_gateway_method_response.options.id,
      aws_api_gateway_integration_response.options.id,
      aws_api_gateway_method.any.id,
      aws_api_gateway_integration.any_integration.id,
      aws_api_gateway_method_response.any_200.id,
      aws_lambda_permission.apigw.id,
    ]))
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_api_gateway_stage" "prod" {
  stage_name    = "prod"
  deployment_id = aws_api_gateway_deployment.this.id
  description   = "Production"
  rest_api_id   = aws_api_gateway_rest_api.this.id
  tags = {
    Project = "gofuel"
  }
  xray_tracing_enabled = true
}

resource "aws_api_gateway_method_settings" "proxy_throttle" {
  rest_api_id = aws_api_gateway_rest_api.this.id
  stage_name  = aws_api_gateway_stage.prod.stage_name
  method_path = "*/*"

  settings {
    metrics_enabled        = true
    throttling_burst_limit = 50
    throttling_rate_limit  = 20
  }
}

# API Gateway: custom domain mapping

data "aws_api_gateway_domain_name" "this" {
  domain_name = "api.seanboaden.dev"
}

resource "aws_api_gateway_base_path_mapping" "gofuel" {
  api_id      = aws_api_gateway_rest_api.this.id
  stage_name  = aws_api_gateway_stage.prod.stage_name
  domain_name = data.aws_api_gateway_domain_name.this.domain_name
  base_path   = "gofuel"
}

# API: Lambda + IAM

resource "aws_iam_role" "api" {
  name = "gofuel-api-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role_policy" "api" {
  name = "gofuel-api-policy"
  role = aws_iam_role.api.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/gofuel*:*"
      },
      {
        Effect = "Allow"
        Action = ["dynamodb:Query"]
        Resource = [
          aws_dynamodb_table.public.arn,
          "${aws_dynamodb_table.public.arn}/index/*",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter",
        ]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/gofuel/*"
      },
      {
        Effect = "Allow"
        Action = [
          "athena:StartQueryExecution",
          "athena:GetQueryExecution",
          "athena:GetQueryResults",
          "athena:GetWorkGroup",
          "athena:StopQueryExecution",
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "glue:GetDatabase",
          "glue:GetDatabases",
          "glue:GetPartition",
          "glue:GetPartitions",
          "glue:GetTable",
          "glue:GetTables",
        ]
        Resource = [
          "arn:aws:glue:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:catalog",
          "arn:aws:glue:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:database/gofuel",
          "arn:aws:glue:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:table/gofuel/*",
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:ListBucket",
        ]
        Resource = [
          aws_s3_bucket.firehose.arn,
          "${aws_s3_bucket.firehose.arn}/*",
        ]
      },
      {
        Effect = "Allow"
        Action = ["s3:PutObject"]
        Resource = [
          "${aws_s3_bucket.firehose.arn}/history/*",
          "${aws_s3_bucket.firehose.arn}/cache/*",
        ]
      },
    ]
  })
}

resource "aws_lambda_function" "this" {
  architectures = ["x86_64"]
  description   = "GoFuel API"
  function_name = "gofuel"
  image_uri     = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com/gofuel:latest"
  memory_size   = 128
  package_type  = "Image"
  role          = aws_iam_role.api.arn
  runtime       = null
  timeout       = 30

  environment {
    variables = {
      DDB_TABLE_STATIONS = aws_dynamodb_table.public.name
      BASE_PATH          = var.base_path
      ENVIRONMENT        = "production"
      S3_BUCKET = aws_s3_bucket.firehose.bucket
    }
  }

  ephemeral_storage {
    size = 512
  }

  logging_config {
    log_format = "Text"
    log_group  = "/aws/lambda/gofuel"
  }

  tracing_config {
    mode = "Active"
  }

  lifecycle {
    ignore_changes = [image_uri]
  }

  tags = {
    Project = "gofuel"
  }
}

# Cron: Lambda + IAM + EventBridge Scheduler

resource "aws_iam_role" "cron" {
  name = "gofuel-cron-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role_policy" "cron" {
  name = "gofuel-cron-policy"
  role = aws_iam_role.cron.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/lambda/gofuel-cron*:*"
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:BatchWriteItem",
        ]
        Resource = aws_dynamodb_table.public.arn
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
        ]
        Resource = aws_dynamodb_table.auth.arn
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter",
        ]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/gofuel/*"
      },
      {
        Effect   = "Allow"
        Action   = ["firehose:PutRecord", "firehose:PutRecordBatch"]
        Resource = aws_kinesis_firehose_delivery_stream.stations.arn
      },
    ]
  })
}

resource "aws_lambda_function" "cron" {
  architectures = ["x86_64"]
  description   = "GoFuel cron refresh"
  function_name = "gofuel-cron"
  image_uri     = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com/gofuel:cron-latest"
  memory_size   = 256
  package_type  = "Image"
  role          = aws_iam_role.cron.arn
  runtime       = null
  timeout       = 300

  environment {
    variables = {
      DDB_TABLE_STATIONS = aws_dynamodb_table.public.name
      DDB_TABLE_AUTH     = aws_dynamodb_table.auth.name
      FIREHOSE_STREAM    = aws_kinesis_firehose_delivery_stream.stations.name
      ENVIRONMENT        = "production"
    }
  }

  ephemeral_storage {
    size = 512
  }

  logging_config {
    log_format = "Text"
    log_group  = "/aws/lambda/gofuel-cron"
  }

  tracing_config {
    mode = "Active"
  }

  lifecycle {
    ignore_changes = [image_uri]
  }

  tags = {
    Project = "gofuel"
  }
}

resource "aws_scheduler_schedule_group" "gofuel" {
  name = "gofuel"

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role" "scheduler" {
  name = "gofuel-scheduler-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "scheduler.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role_policy" "scheduler" {
  name = "gofuel-scheduler-policy"
  role = aws_iam_role.scheduler.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "lambda:InvokeFunction"
      Resource = aws_lambda_function.cron.arn
    }]
  })
}

resource "aws_scheduler_schedule" "wa_tomorrow" {
  name       = "gofuel-cron-wa-tomorrow"
  group_name = aws_scheduler_schedule_group.gofuel.name
  state      = "ENABLED"

  schedule_expression          = "cron(0 16 * * ? *)"
  schedule_expression_timezone = "Australia/Perth"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_lambda_function.cron.arn
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ provider = "wa", day = "tomorrow" })
  }
}

resource "aws_scheduler_schedule" "nsw_tas" {
  name       = "gofuel-cron-nsw-tas"
  group_name = aws_scheduler_schedule_group.gofuel.name
  state      = "ENABLED"

  schedule_expression          = "cron(0 5,9,12,15,17 * * ? *)"
  schedule_expression_timezone = "Australia/Perth"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_lambda_function.cron.arn
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ provider = "nsw_tas", day = "" })
  }
}

resource "aws_scheduler_schedule" "sa_qld" {
  name       = "gofuel-cron-sa-qld"
  group_name = aws_scheduler_schedule_group.gofuel.name
  state      = "ENABLED"

  schedule_expression          = "cron(0 5,9,12,15,17 * * ? *)"
  schedule_expression_timezone = "Australia/Perth"

  flexible_time_window {
    mode = "OFF"
  }

  target {
    arn      = aws_lambda_function.cron.arn
    role_arn = aws_iam_role.scheduler.arn
    input    = jsonencode({ provider = "sa_qld", day = "" })
  }
}

# Firehose -> S3: archived station records, partitioned by event date

resource "aws_s3_bucket" "firehose" {
  bucket = "gofuel-firehose-historical"

  tags = {
    Project = "gofuel"
  }
}

resource "aws_s3_bucket_ownership_controls" "firehose" {
  bucket = aws_s3_bucket.firehose.id

  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_public_access_block" "firehose" {
  bucket                  = aws_s3_bucket.firehose.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "firehose" {
  bucket = aws_s3_bucket.firehose.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "firehose" {
  bucket = aws_s3_bucket.firehose.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "firehose" {
  bucket = aws_s3_bucket.firehose.id

  rule {
    id     = "data-transition"
    status = "Enabled"

    filter {
      prefix = ""
    }

    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }
  }

  rule {
    id     = "errors-expire"
    status = "Enabled"

    filter {
      prefix = "errors/"
    }

    expiration {
      days = 7
    }
  }
}

resource "aws_cloudwatch_log_group" "firehose" {
  name              = "/aws/kinesisfirehose/gofuel-firehose"
  retention_in_days = 14

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role" "firehose" {
  name = "gofuel-firehose-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "firehose.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = {
    Project = "gofuel"
  }
}

resource "aws_iam_role_policy" "firehose" {
  name = "gofuel-firehose-policy"
  role = aws_iam_role.firehose.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:AbortMultipartUpload",
          "s3:GetBucketLocation",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:ListBucketMultipartUploads",
          "s3:PutObject",
        ]
        Resource = [
          aws_s3_bucket.firehose.arn,
          "${aws_s3_bucket.firehose.arn}/*",
        ]
      },
      {
        Effect   = "Allow"
        Action   = ["logs:PutLogEvents"]
        Resource = "${aws_cloudwatch_log_group.firehose.arn}:*"
      },
    ]
  })
}

resource "aws_kinesis_firehose_delivery_stream" "stations" {
  name        = "gofuel-stations"
  destination = "extended_s3"

  server_side_encryption {
    enabled = true
  }

  extended_s3_configuration {
    role_arn            = aws_iam_role.firehose.arn
    bucket_arn          = aws_s3_bucket.firehose.arn
    prefix              = "!{partitionKeyFromQuery:date}/"
    error_output_prefix = "errors/!{firehose:error-output-type}/!{timestamp:yyyy/MM/dd}/"
    compression_format  = "GZIP"
    file_extension      = ".ndjson.gz"

    buffering_size     = 64
    buffering_interval = 60

    dynamic_partitioning_configuration {
      enabled        = true
      retry_duration = 300
    }

    processing_configuration {
      enabled = true

      processors {
        type = "MetadataExtraction"
        parameters {
          parameter_name  = "MetadataExtractionQuery"
          parameter_value = "{date:(.date|gsub(\"-\";\"/\"))}"
        }
        parameters {
          parameter_name  = "JsonParsingEngine"
          parameter_value = "JQ-1.6"
        }
      }
    }

    cloudwatch_logging_options {
      enabled         = true
      log_group_name  = aws_cloudwatch_log_group.firehose.name
      log_stream_name = "DestinationDelivery"
    }
  }

  tags = {
    Project = "gofuel"
  }
}

# Athena historical stuff

resource "aws_athena_workgroup" "gofuel" {
  name = "gofuel"

  configuration {
    enforce_workgroup_configuration    = true
    publish_cloudwatch_metrics_enabled = true
    result_configuration {
      output_location = "s3://${aws_s3_bucket.firehose.bucket}/history/"
    }
  }

  tags = {
    Project = "gofuel"
  }
}

resource "aws_glue_catalog_database" "this" {
  name = "gofuel"
}

resource "aws_glue_catalog_table" "stations_raw" {
  name          = "stations_raw"
  database_name = aws_glue_catalog_database.this.name
  table_type    = "EXTERNAL_TABLE"

  parameters = {
    EXTERNAL                      = "TRUE"
    "projection.enabled"          = "true"
    "projection.dt.type"          = "date"
    "projection.dt.range"         = "2024/01/01,NOW"
    "projection.dt.format"        = "yyyy/MM/dd"
    "projection.dt.interval"      = "1"
    "projection.dt.interval.unit" = "DAYS"
    "storage.location.template"   = "s3://${aws_s3_bucket.firehose.bucket}/$${dt}/"
    "compression.type"            = "GZIP"
  }

  storage_descriptor {
    location      = "s3://${aws_s3_bucket.firehose.bucket}/"
    input_format  = "org.apache.hadoop.mapred.TextInputFormat"
    output_format = "org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat"

    ser_de_info {
      name                  = "stations-serde"
      serialization_library = "org.openx.data.jsonserde.JsonSerDe"
    }

    columns {
      name = "region_geohash"
      type = "string"
    }
    columns {
      name = "town_geohash"
      type = "string"
    }
    columns {
      name = "sub_region_geohash"
      type = "string"
    }
    columns {
      name = "station_id"
      type = "string"
    }
    columns {
      name = "title"
      type = "string"
    }
    columns {
      name = "brand"
      type = "string"
    }
    columns {
      name = "address"
      type = "string"
    }
    columns {
      name = "latitude"
      type = "double"
    }
    columns {
      name = "longitude"
      type = "double"
    }
    columns {
      name = "ulp91"
      type = "float"
    }
    columns {
      name = "ulp95"
      type = "float"
    }
    columns {
      name = "ulp98"
      type = "float"
    }
    columns {
      name = "diesel"
      type = "float"
    }
    columns {
      name = "date"
      type = "string"
    }
  }

  partition_keys {
    name = "dt"
    type = "string"
  }
}

resource "aws_athena_named_query" "stations_view" {
  name        = "stations (deduped view)"
  workgroup   = aws_athena_workgroup.gofuel.name
  database    = aws_glue_catalog_database.this.name
  description = "Consolidated view of gofuel.stations_raw"

  query = <<-SQL
    CREATE OR REPLACE VIEW gofuel.stations AS
    WITH ranked AS (
      SELECT
        dt, date, station_id, brand, title, address,
        latitude, longitude, ulp91, ulp95, ulp98, diesel,
        region_geohash, sub_region_geohash, town_geohash,
        ROW_NUMBER() OVER (
          PARTITION BY station_id, date
          ORDER BY "$path" DESC
        ) AS rn
      FROM gofuel.stations_raw
    )
    SELECT
      dt, date, station_id, brand, title, address,
      latitude, longitude, ulp91, ulp95, ulp98, diesel,
      region_geohash, sub_region_geohash, town_geohash
    FROM ranked
    WHERE rn = 1;
  SQL
}

# IAM Role for github oidc ci/cd

data "aws_ecr_repository" "gofuel" {
  name = "gofuel"
}

data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

data "aws_iam_policy_document" "github_actions_assume_role" {
  statement {
    sid     = "GithubOidcAssumeRole"
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [data.aws_iam_openid_connect_provider.github.arn]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.gh_repo}:environment:production"]
    }
  }
}

resource "aws_iam_role" "github_actions_role" {
  name               = "gofuel-cicd-deploy-role"
  assume_role_policy = data.aws_iam_policy_document.github_actions_assume_role.json
}

data "aws_iam_policy_document" "deployment_policy" {
  statement {
    sid       = "AllowLambdaUpdate"
    actions   = ["lambda:UpdateFunctionCode"]
    effect    = "Allow"
    resources = [aws_lambda_function.this.arn, aws_lambda_function.cron.arn]
  }
  statement {
    sid    = "AllowEcrAuth"
    effect = "Allow"
    actions = [
      "ecr:GetAuthorizationToken",
    ]
    resources = ["*"]
  }
  statement {
    sid    = "AllowEcrPush"
    effect = "Allow"
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:GetDownloadUrlForLayer",
      "ecr:GetRepositoryPolicy",
      "ecr:DescribeRepositories",
      "ecr:ListImages",
      "ecr:DescribeImages",
      "ecr:BatchGetImage",
      "ecr:InitiateLayerUpload",
      "ecr:UploadLayerPart",
      "ecr:CompleteLayerUpload",
      "ecr:PutImage",
    ]
    resources = [data.aws_ecr_repository.gofuel.arn]
  }
}

resource "aws_iam_role_policy" "deployment" {
  name   = "gofuel-cicd-deploy-policy"
  role   = aws_iam_role.github_actions_role.id
  policy = data.aws_iam_policy_document.deployment_policy.json
}
