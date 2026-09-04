#!/usr/bin/env bash
#
# HTTP contract tests for the resources and opportunities routes.
#
# Start the API first (npm start), then:
#   ./scripts/api-contract.sh
#   BASE_URL=https://api.codecoogs.com/v1 ./scripts/api-contract.sh
#
# Creates rows tagged with a unique marker and deletes them on the way out.

set -u

BASE_URL="${BASE_URL:-http://localhost:3000/v1}"
MARKER="contract-test-$$-$(date +%s)"

passed=0
failed=0

status_of() {
	curl -s -o /dev/null -w '%{http_code}' "$@"
}

body_of() {
	curl -s "$@"
}

check() {
	local desc="$1" want="$2" got="$3"
	if [ "$want" = "$got" ]; then
		printf '  ok    %s\n' "$desc"
		passed=$((passed + 1))
	else
		printf '  FAIL  %s (want %s, got %s)\n' "$desc" "$want" "$got"
		failed=$((failed + 1))
	fi
}

contains() {
	local desc="$1" needle="$2" haystack="$3"
	case "$haystack" in
	*"$needle"*)
		printf '  ok    %s\n' "$desc"
		passed=$((passed + 1))
		;;
	*)
		printf '  FAIL  %s (missing %s)\n' "$desc" "$needle"
		failed=$((failed + 1))
		;;
	esac
}

extract_id() {
	printf '%s' "$1" | sed -n 's/.*"id":"\([0-9a-fA-F-]\{36\}\)".*/\1/p' | head -1
}

echo "Contract tests against $BASE_URL"
echo "Marker: $MARKER"

# ---------------------------------------------------------------- resources
echo
echo "resources"

check "GET list returns 200" 200 "$(status_of "$BASE_URL/resources")"
contains "GET list is a success envelope" '"success":true' "$(body_of "$BASE_URL/resources")"
check "GET list filtered by website_viewable returns 200" 200 \
	"$(status_of "$BASE_URL/resources?website_viewable=true")"
check "GET list filtered by is_active returns 200" 200 \
	"$(status_of "$BASE_URL/resources?is_active=true")"

check "GET rejects non-boolean website_viewable" 400 \
	"$(status_of "$BASE_URL/resources?website_viewable=yes")"
check "GET rejects non-boolean is_active" 400 \
	"$(status_of "$BASE_URL/resources?is_active=2")"
check "GET rejects empty website_viewable" 400 \
	"$(status_of "$BASE_URL/resources?website_viewable=")"

check "POST rejects malformed JSON" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"broken"' "$BASE_URL/resources")"
check "POST rejects a missing title" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"category":"Workshops","link_url":"https://example.com/a.pdf"}' "$BASE_URL/resources")"
check "POST rejects a missing category" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"x","link_url":"https://example.com/a.pdf"}' "$BASE_URL/resources")"
check "POST rejects a missing link_url" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"x","category":"Workshops"}' "$BASE_URL/resources")"
check "POST rejects a whitespace-only title" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"   ","category":"Workshops","link_url":"https://example.com/a.pdf"}' "$BASE_URL/resources")"

check "PATCH on the collection is not allowed" 405 \
	"$(status_of -X PATCH "$BASE_URL/resources")"

check "POST creates a resource" 200 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d "{\"title\":\"$MARKER\",\"category\":\"$MARKER\",\"link_url\":\"https://example.com/a.pdf\"}" \
		"$BASE_URL/resources")"

created=$(body_of "$BASE_URL/resources?category=$MARKER")
resource_id=$(extract_id "$created")

if [ -z "$resource_id" ]; then
	echo "  FAIL  could not read back the created resource"
	failed=$((failed + 1))
else
	echo "  ..    created resource $resource_id"
	contains "created row defaults website_viewable to false" '"website_viewable":false' "$created"
	contains "created row defaults is_active to true" '"is_active":true' "$created"
	contains "created row stamps created_at" '"created_at"' "$created"

	check "GET by id returns 200" 200 "$(status_of "$BASE_URL/resources?id=$resource_id")"
	contains "GET by id returns the row" "$MARKER" "$(body_of "$BASE_URL/resources?id=$resource_id")"

	check "PUT rejects an invalid body" 400 \
		"$(status_of -X PUT -H 'Content-Type: application/json' \
			-d '{"category":"Workshops","link_url":"https://example.com/a.pdf"}' \
			"$BASE_URL/resources?id=$resource_id")"

	check "PUT updates the resource" 200 \
		"$(status_of -X PUT -H 'Content-Type: application/json' \
			-d "{\"title\":\"$MARKER updated\",\"category\":\"$MARKER\",\"link_url\":\"https://example.com/b.pdf\",\"website_viewable\":true}" \
			"$BASE_URL/resources?id=$resource_id")"

	updated=$(body_of "$BASE_URL/resources?id=$resource_id")
	contains "PUT persisted the new title" "$MARKER updated" "$updated"
	contains "PUT flipped website_viewable" '"website_viewable":true' "$updated"
	contains "the updated_at trigger fired" '"updated_at"' "$updated"

	check "DELETE removes the resource" 200 \
		"$(status_of -X DELETE "$BASE_URL/resources?id=$resource_id")"
	check "GET after DELETE returns an empty list" '{"success":true}' \
		"$(body_of "$BASE_URL/resources?id=$resource_id")"
fi

# ------------------------------------------------------------ opportunities
echo
echo "opportunities"

check "GET list returns 200" 200 "$(status_of "$BASE_URL/opportunities")"
contains "GET list is a success envelope" '"success":true' "$(body_of "$BASE_URL/opportunities")"
check "GET list filtered by website_viewable returns 200" 200 \
	"$(status_of "$BASE_URL/opportunities?website_viewable=true")"

check "GET rejects non-boolean website_viewable" 400 \
	"$(status_of "$BASE_URL/opportunities?website_viewable=maybe")"
check "GET rejects non-boolean is_active" 400 \
	"$(status_of "$BASE_URL/opportunities?is_active=2")"

check "POST rejects malformed JSON" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"broken"' "$BASE_URL/opportunities")"
check "POST rejects a missing title" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"term":"Spring 2026","link_url":"https://forms.gle/abc"}' "$BASE_URL/opportunities")"
check "POST rejects neither link_url nor linked_form_id" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"x","term":"Spring 2026"}' "$BASE_URL/opportunities")"
check "POST rejects both link_url and linked_form_id" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"x","link_url":"https://forms.gle/abc","linked_form_id":"c4e11505-5555-42f9-9f48-678a0abf55c6"}' \
		"$BASE_URL/opportunities")"
check "POST rejects a whitespace-only title" 400 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d '{"title":"  ","link_url":"https://forms.gle/abc"}' "$BASE_URL/opportunities")"

check "PATCH on the collection is not allowed" 405 \
	"$(status_of -X PATCH "$BASE_URL/opportunities")"

check "POST creates an opportunity" 200 \
	"$(status_of -X POST -H 'Content-Type: application/json' \
		-d "{\"title\":\"$MARKER\",\"link_url\":\"https://forms.gle/abc\",\"category\":\"Club Role\",\"term\":\"Spring 2026\"}" \
		"$BASE_URL/opportunities")"

# The list is unfiltered, so isolate the row this run created by its marker title.
created=$(body_of "$BASE_URL/opportunities")
opportunity_id=$(printf '%s' "$created" |
	sed -n "s/.*{\"id\":\"\([0-9a-fA-F-]\{36\}\)\",\"title\":\"$MARKER\".*/\1/p" | head -1)

if [ -z "$opportunity_id" ]; then
	echo "  FAIL  could not read back the created opportunity"
	failed=$((failed + 1))
else
	echo "  ..    created opportunity $opportunity_id"
	contains "created row defaults website_viewable to false" '"website_viewable":false' "$created"

	check "GET by id returns 200" 200 "$(status_of "$BASE_URL/opportunities?id=$opportunity_id")"
	contains "GET by id returns the row" "$MARKER" "$(body_of "$BASE_URL/opportunities?id=$opportunity_id")"

	check "PUT rejects an invalid body" 400 \
		"$(status_of -X PUT -H 'Content-Type: application/json' \
			-d '{"term":"Spring 2026","link_url":"https://forms.gle/abc"}' \
			"$BASE_URL/opportunities?id=$opportunity_id")"

	check "PUT updates the opportunity" 200 \
		"$(status_of -X PUT -H 'Content-Type: application/json' \
			-d "{\"title\":\"$MARKER updated\",\"link_url\":\"https://forms.gle/abc\",\"website_viewable\":true}" \
			"$BASE_URL/opportunities?id=$opportunity_id")"

	updated=$(body_of "$BASE_URL/opportunities?id=$opportunity_id")
	contains "PUT persisted the new title" "$MARKER updated" "$updated"
	contains "PUT stamped updated_at" '"updated_at"' "$updated"

	check "DELETE removes the opportunity" 200 \
		"$(status_of -X DELETE "$BASE_URL/opportunities?id=$opportunity_id")"
fi

echo
echo "passed: $passed  failed: $failed"
[ "$failed" -eq 0 ]
