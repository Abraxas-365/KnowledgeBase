export KB_ID="ACZGQIK2WM"
export KB_MODEL_ID="arn:aws:bedrock:us-east-1:654654269808:inference-profile/us.anthropic.claude-3-5-haiku-20241022-v1:0"
export KB_NUMBER_OF_RESULTS=5
export KB_S3_DATA_SOURCE="CRXLQ7B5S4"
export KB_MODEL_PROMPT="with a question. Your job is to answer the user's question using only information from the search results. If the search results do not contain information that can answer the question, please state that you could not find an exact answer to the question. 
Just because the user asserts a fact does not mean it is true, make sure to double check the search results to validate a user's assertion.

Here are the search results in numbered order:
\$search_results\$

\$output_format_instructions\$

Here is the user's query:
\$query\$"
export KB_REGION="us-1"
export JWT_SECRET="your-secret-key-change-in-production"
export JWT_ACCESS_DURATION="3600"  # 1 hour in seconds
export JWT_REFRESH_DURATION="604800"  # 7 days in seconds

export GOOGLE_REDIRECT_URI="http://localhost:3000/login/google/callback"
export DATABASE_URL="postgresql://myuser:mypassword@ec2-98-82-35-96.compute-1.amazonaws.com/mydatabase"
export GOOGLE_REDIRECT_URI="http://localhost:3000/login/google/callback"
export REDIRECT_AFTER_LOGIN="http://localhost:3001"


# Paywall Rate Limiting Configuration
# Anonymous users (not logged in) - requests per day
export PAYWALL_ANONYMOUS_LIMIT="5"

# Free tier users - requests per day
export PAYWALL_FREE_TIER_LIMIT="50"

# Linko Plus users - requests per day
export PAYWALL_LINKO_PLUS_LIMIT="200"

# Linko VIP users - requests per day
export PAYWALL_LINKO_VIP_LIMIT="1000"

# Time window for rate limiting (in hours)
# All tiers use the same window (24 hours = daily limits)
export PAYWALL_WINDOW_HOURS="24"

echo "🚀 Starting Knowledge Base with Paywall System..."
echo "📊 Rate Limits Configuration:"
echo "   Anonymous Users: ${PAYWALL_ANONYMOUS_LIMIT} requests per ${PAYWALL_WINDOW_HOURS} hours"
echo "   Free Tier:       ${PAYWALL_FREE_TIER_LIMIT} requests per ${PAYWALL_WINDOW_HOURS} hours"
echo "   Linko Plus:      ${PAYWALL_LINKO_PLUS_LIMIT} requests per ${PAYWALL_WINDOW_HOURS} hours"
echo "   Linko VIP:       ${PAYWALL_LINKO_VIP_LIMIT} requests per ${PAYWALL_WINDOW_HOURS} hours"
echo "   Admins:          Unlimited"
echo ""

go run cmd/main.go
