#!/bin/bash

API_URL="http://localhost:8080/api/v1/questions"
AUTH_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJlbWFpbCI6ImFkbWluQGV4YW1wbGUuY29tIiwidXNlcm5hbWUiOiJhZG1pbiIsInJvbGUiOiJhZG1pbiIsInN1YiI6IjIiLCJleHAiOjE3NTQ5NzIwMjcsIm5iZiI6MTc1NDk3MTEyNywiaWF0IjoxNzU0OTcxMTI3fQ.AVTx8FctSimhoOb9SPOjNAtaxr4ZvAoNqfqqVtvOCAA"

TYPES=("multiple_choice" "true_false")
DIFFICULTIES=("easy" "medium" "hard")
TAGS=("golang" "javascript" "devops" "linux" "database")

for i in $(seq 1 100); do
	TITLE="Question $i: $(openssl rand -hex 4)"
	CONTENT="This is the content for question $i"
	TYPE=${TYPES[$RANDOM % ${#TYPES[@]}]}
	DIFFICULTY=${DIFFICULTIES[$RANDOM % ${#DIFFICULTIES[@]}]}
	TAG1=${TAGS[$RANDOM % ${#TAGS[@]}]}
	TAG2=${TAGS[$RANDOM % ${#TAGS[@]}]}
	POINTS=$((RANDOM % 5 + 1))
	TIME_LIMIT=$((RANDOM % 120 + 30))

	if [[ "$TYPE" == "multiple_choice" ]]; then
		CORRECT_INDEX=$((RANDOM % 4))
		OPTIONS=""
		for j in 0 1 2 3; do
			ID=$(printf "\\$(printf '%03o' $((97 + j)))") # a,b,c,d
			TEXT="Option $(tr '[:lower:]' '[:upper:]' <<<$ID)"
			IS_CORRECT="false"
			if [[ $j -eq $CORRECT_INDEX ]]; then
				IS_CORRECT="true"
			fi
			OPTIONS="$OPTIONS{\"id\": \"$ID\", \"text\": \"$TEXT\", \"is_correct\": $IS_CORRECT},"
		done
		OPTIONS="[${OPTIONS%,}]"
	else
		CORRECT_INDEX=$((RANDOM % 2))
		OPTIONS=""
		for j in 0 1; do
			ID=$(printf "\\$(printf '%03o' $((97 + j)))") # a,b
			TEXT=$([[ $j -eq 0 ]] && echo "True" || echo "False")
			IS_CORRECT="false"
			if [[ $j -eq $CORRECT_INDEX ]]; then
				IS_CORRECT="true"
			fi
			OPTIONS="$OPTIONS{\"id\": \"$ID\", \"text\": \"$TEXT\", \"is_correct\": $IS_CORRECT},"
		done
		OPTIONS="[${OPTIONS%,}]"
	fi

	read -r -d '' JSON_DATA <<EOF
{
  "title": "$TITLE",
  "content": "$CONTENT",
  "type": "$TYPE",
  "difficulty": "$DIFFICULTY",
  "options": $OPTIONS,
  "tags": ["$TAG1", "$TAG2"],
  "points": $POINTS,
  "time_limit": $TIME_LIMIT,
  "explanation": "Auto-generated explanation for $TITLE"
}
EOF

	echo "Seeding question $i ($TYPE)..."
	curl -s --location "$API_URL" \
		--header "Content-Type: application/json" \
		--header "Authorization: Bearer $AUTH_TOKEN" \
		--data "$JSON_DATA" >/dev/null
done

echo "✅ Seeded 100 random questions successfully with correct options count!"
