.PHONY: cron cron-wa cron-wa-tomorrow cron-nsw cron-sa

cron:
	go run ./cmd/cron $(p) $(day)

cron-wa:
	go run ./cmd/cron wa

cron-wa-tomorrow:
	go run ./cmd/cron wa tomorrow

cron-nsw:
	go run ./cmd/cron nsw_tas

cron-sa:
	go run ./cmd/cron sa_qld