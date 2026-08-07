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

# ---------------------------------------------------------------------------
# SSM Parameter Store: provider API keys (plaintext, values set out-of-band)
# ---------------------------------------------------------------------------

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
}
#
# resource "aws_api_gateway_resource" "root" {
#   rest_api_id = aws_api_gateway_rest_api.this.id
#   path_part   = ""
#   parent_id   = ""
# }
#
# # __generated__ by Terraform from "njwn9hkiph/tv01xg"
# resource "aws_api_gateway_resource" "proxy" {
#   rest_api_id = aws_api_gateway_rest_api.this.id
#   path_part   = "{proxy+}"
#   parent_id   = aws_api_gateway_resource.root.id
# }
#
# # __generated__ by Terraform from "njwn9hkiph/tv01xg/ANY"
# resource "aws_api_gateway_method" "any" {
#   api_key_required     = false
#   authorization        = "NONE"
#   authorization_scopes = []
#   authorizer_id        = null
#   http_method          = "ANY"
#   operation_name       = null
#   request_models       = {}
#   request_parameters = {
#     "method.request.path.proxy" = true
#   }
#   request_validator_id = null
#   resource_id          = aws_api_gateway_resource.proxy.id
#   rest_api_id          = aws_api_gateway_rest_api.this.id
# }
#
# # __generated__ by Terraform from "njwn9hkiph/c6gk9famag"
#
# # __generated__ by Terraform from "njwn9hkiph/tv01xg/ANY"
# resource "aws_api_gateway_integration" "any_integration" {
#   http_method             = "ANY"
#   integration_http_method = "POST"
#   passthrough_behavior    = "WHEN_NO_MATCH"
#   request_parameters      = {}
#   request_templates       = {}
#   resource_id             = aws_api_gateway_resource.proxy.id
#   rest_api_id             = aws_api_gateway_rest_api.this.id
#   timeout_milliseconds    = 15000
#   type                    = "AWS_PROXY"
#   uri                     = "arn:aws:apigateway:${data.aws_region.current.name}:lambda:path/2015-03-31/functions/arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:gofuel/invocations"
# }
#
# # __generated__ by Terraform from "njwn9hkiph/prod"
# resource "aws_api_gateway_stage" "prod" {
#   stage_name           = "prod"
#   description          = "Production"
#   deployment_id        = "udi98p"
#   rest_api_id          = aws_api_gateway_rest_api.this.id
#   tags                 = {}
#   tags_all             = {}
#   variables            = {}
#   xray_tracing_enabled = false
# }

# ---------------------------------------------------------------------------
# API: Lambda + IAM (image_uri managed by CI; config managed by Terraform)
# ---------------------------------------------------------------------------

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
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = aws_dynamodb_table.public.arn
      },
      {
        Effect = "Allow"
        Action = [
          "ssm:GetParameters",
          "ssm:GetParameter",
        ]
        Resource = "arn:aws:ssm:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:parameter/gofuel/*"
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

# ---------------------------------------------------------------------------
# Cron: Lambda + IAM + EventBridge Scheduler
# ---------------------------------------------------------------------------

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

# ---------------------------------------------------------------------------
# Firehose -> S3: archived station records, partitioned by event date
# ---------------------------------------------------------------------------

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

    expiration {
      days = 365
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
