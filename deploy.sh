#!/usr/bin/env bash
set -e

PROJECT_ID="project-55bf7b12-d66c-4cc5-aa0"
SERVICE="nim-stick"
REGION="asia-south1"             # Mumbai
IMAGE="gcr.io/$PROJECT_ID/$SERVICE"

echo "▶ Building and deploying $SERVICE to Cloud Run ($REGION)..."

gcloud run deploy "$SERVICE" \
  --source . \
  --project "$PROJECT_ID" \
  --region "$REGION" \
  --platform managed \
  --allow-unauthenticated \
  --port 8080 \
  --memory 256Mi \
  --cpu 1 \
  --min-instances 0 \
  --max-instances 10 \
  --timeout 3600 \
  --set-env-vars PORT=8080

echo ""
echo "✅ Deployed. Your app URL:"
gcloud run services describe "$SERVICE" \
  --project "$PROJECT_ID" \
  --region "$REGION" \
  --format "value(status.url)"
