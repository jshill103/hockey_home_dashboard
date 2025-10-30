#!/bin/bash
# Kubernetes Deployment Update Script
# This script updates your deployment to include fsGroup for PVC permissions

set -e

DEPLOYMENT_NAME="${DEPLOYMENT_NAME:-hockey-dashboard}"
NAMESPACE="${NAMESPACE:-default}"

echo "🔧 Updating Kubernetes deployment: $DEPLOYMENT_NAME in namespace: $NAMESPACE"

# Check if deployment exists
if ! kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" &>/dev/null; then
    echo "❌ Error: Deployment '$DEPLOYMENT_NAME' not found in namespace '$NAMESPACE'"
    echo "   Please set DEPLOYMENT_NAME and NAMESPACE environment variables"
    exit 1
fi

# Patch the deployment to add securityContext with fsGroup
echo "📝 Adding securityContext with fsGroup: 1001..."
kubectl patch deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" --type json -p='
[
  {
    "op": "add",
    "path": "/spec/template/spec/securityContext",
    "value": {
      "fsGroup": 1001,
      "runAsNonRoot": true,
      "runAsUser": 1001,
      "runAsGroup": 1001
    }
  }
]'

echo "✅ Security context added successfully"
echo ""
echo "🔄 Rolling out new image..."
kubectl set image deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" \
  hockey-dashboard=jshillingburg/hockey_home_dashboard:latest || \
  kubectl set image deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" \
  "*/hockey-dashboard=jshillingburg/hockey_home_dashboard:latest"

echo ""
echo "⏳ Waiting for rollout to complete..."
kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout=5m

echo ""
echo "✅ Deployment updated successfully!"
echo ""
echo "📋 Check pod logs to verify directory initialization:"
echo "   kubectl logs -n $NAMESPACE -l app=$DEPLOYMENT_NAME --tail=50"
echo ""
echo "   You should see: '✅ Data directories initialized successfully'"

