run:
	go run ./cmd/main.go \
    		--path="./migrations/postgresql" \
    		--table="migrations_history" \
    		--major=0 \
    		--minor=0