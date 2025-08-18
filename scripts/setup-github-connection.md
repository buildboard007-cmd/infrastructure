# GitHub Connection Setup

The GitHub connection ARN in your config appears to be from a different AWS account (909408398654).
You need to create a new connection in your Tools account (401448503050).

## Steps to Create GitHub Connection:

1. **Login to Tools Account AWS Console** (401448503050)

2. **Navigate to Developer Tools > Connections**
   - Go to: https://us-west-2.console.aws.amazon.com/codesuite/settings/connections
   - Make sure you're in us-west-2 region (as per your config)

3. **Create New Connection**
   - Click "Create connection"
   - Select "GitHub"
   - Give it a name: "infrastructure-github"
   - Click "Connect to GitHub"
   - Authorize AWS Connector for GitHub
   - Select your GitHub organization or account
   - Install and authorize the GitHub Apps

4. **Copy the Connection ARN**
   - After creation, copy the full ARN
   - It will look like: `arn:aws:codeconnections:us-west-2:401448503050:connection/xxxxxx`

5. **Update config/config.ts**
   - Replace line 20 with your new connection ARN

## Alternative: Using AWS CLI

```bash
# Create connection (requires browser interaction)
aws codeconnections create-connection \
  --provider-type GitHub \
  --connection-name infrastructure-github \
  --region us-west-2 \
  --profile tools

# List connections to get the ARN
aws codeconnections list-connections \
  --region us-west-2 \
  --profile tools \
  --query 'Connections[?ConnectionName==`infrastructure-github`].ConnectionArn' \
  --output text
```

Note: You'll still need to complete the GitHub authorization through the console.