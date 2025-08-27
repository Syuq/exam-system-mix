#!/usr/bin/env bash
set -euo pipefail

XML_FILE="vidu cau hoi.xml"
API_URL="http://localhost:8080/api/v1/questions"
AUTH_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJlbWFpbCI6ImFkbWluQGV4YW1wbGUuY29tIiwidXNlcm5hbWUiOiJhZG1pbiIsInJvbGUiOiJhZG1pbiIsInN1YiI6IjIiLCJleHAiOjE3NTYyNzUwMjUsIm5iZiI6MTc1NjI3NDEyNSwiaWF0IjoxNzU2Mjc0MTI1fQ.Obx9L04_sYSljdUp5LV70mwIL2HpI1REHlegXd2BZmw"

# count questions
count=$(xmlstarlet sel -t -v "count(/quiz/question)" "$XML_FILE")

for i in $(seq 1 "$count"); do
	title=$(xmlstarlet sel -t -v "/quiz/question[$i]/name/text" "$XML_FILE")
	content=$(xmlstarlet sel -t -v "/quiz/question[$i]/questiontext/text" "$XML_FILE")

	# extract answers
	answers=$(xmlstarlet sel -t -m "/quiz/question[$i]/answer" \
		-v "concat(substring(text,1,1), '|', normalize-space(text), '|', @fraction)" -n "$XML_FILE" |
		awk -F'|' '{
      is_correct = ($3 == "100") ? "true" : "false";
      printf("{\"id\":\"%s\",\"text\":\"%s\",\"is_correct\":%s}\n", $1, $2, is_correct);
    }' | jq -s '.')

	# build JSON body
	json=$(
		jq -n \
			--arg title "$title" \
			--arg content "$content" \
			--arg type "multiple_choice" \
			--arg difficulty "medium" \
			--argjson options "$answers" \
			'{
      title: $title,
      content: $content,
      type: $type,
      difficulty: $difficulty,
      options: $options,
      tags: ["quiz","import"],
      points: 1,
      time_limit: 60,
      explanation: "Imported from XML"
    }'
	)

	echo ">>> Sending question $i..."
	curl -s -X POST "$API_URL" \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $AUTH_TOKEN" \
		-d "$json"

	echo -e "\n--- Done question $i ---"
done
