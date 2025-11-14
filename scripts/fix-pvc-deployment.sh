#!/bin/bash
# Fix PVC Multi-Attach Issue by Changing Deployment Strategy
# This changes from RollingUpdate to Recreate strategy

set -e

DEPLOYMENT_NAME="${1:-hockey-dashboard}"
NAMESPACE="${2:-hockey-dashboard}"

echo "🔧 Fixing PVC Multi-Attach issue for deployment: $DEPLOYMENT_NAME"
echo "   Namespace: $NAMESPACE"
echo ""

# Check if deployment exists
if ! kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" &>/dev/null; then
    echo "❌ Error: Deployment '$DEPLOYMENT_NAME' not found in namespace '$NAMESPACE'"
    echo ""
    echo "Usage: $0 [deployment-name] [namespace]"
    echo "Example: $0 hockey-dashboard hockey-dashboard"
    exit 1
fi

echo "📋 Current deployment strategy:"
kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.strategy.type}' || echo "Not set (defaults to RollingUpdate)"
echo ""
echo ""

echo "🔄 Changing deployment strategy to 'Recreate'..."
echo "   This will:"
echo "   ✅ Stop old pod completely before starting new one"
echo "   ✅ Eliminate volume attachment conflicts"
echo "   ⚠️  Brief downtime during updates (~5-10 seconds)"
echo ""

# Patch the deployment strategy
kubectl patch deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" --type='json' -p='[
  {
    "op": "replace",
    "path": "/spec/strategy",
    "value": {
      "type": "Recreate"
    }
  }
]'

echo ""
echo "✅ Deployment strategy updated successfully!"
echo ""

echo "📋 New deployment strategy:"
kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.spec.strategy.type}'
echo ""
echo ""

echo "🎯 Testing the fix with a rollout restart..."
kubectl rollout restart deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE"

echo ""
echo "⏳ Waiting for rollout to complete..."
kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n "$NAMESPACE" --timeout=3m

echo ""
echo "✅ Success! PVC issue is fixed."
echo ""
echo "📝 What changed:"
echo "   • Strategy: RollingUpdate → Recreate"
echo "   • Effect: No more 'Multi-Attach error for volume' messages"
echo "   • Trade-off: Brief downtime during updates (acceptable for single-replica deployment)"
echo ""
echo "🔮 Future updates will now:"
echo "   1. Terminate old pod completely"
echo "   2. Wait for volume to detach"
echo "   3. Start new pod with fresh volume mount"
echo ""

