# IMPORTANT INSTRUCTIONS FOR CLAUDE

## 🚨 CRITICAL: DATABASE ACCESS RULE 🚨
**NEVER EVER use bash/node scripts for database queries!**
**ALWAYS use simple natural language to query the database through MCP!**

❌ **FORBIDDEN:** `node -e "const pg = require('pg')..."`
❌ **FORBIDDEN:** `cd mcp-postgres-server && node dist/index.js...`
❌ **FORBIDDEN:** Any direct database connection attempts

✅ **CORRECT:** Just ask: "Show me all tables in project schema"
✅ **CORRECT:** Just ask: "What columns are in project_managers table?"
✅ **CORRECT:** Just ask: "How many users are active?"

The MCP server `postgres-construction` is already running and connected.
IT AUTOMATICALLY HANDLES ALL DATABASE QUERIES IN NATURAL LANGUAGE.

### Database connection details (for reference only):
- Host: appdb.cdwmaay8wkw4.us-east-2.rds.amazonaws.com
- Database: appdb
- User: appdb_admin
- Schemas: iam, project

## Project Structure
- Backend: Go (Golang) - `/src` directory
- Infrastructure: AWS CDK TypeScript - `/lib` directory
- Database: PostgreSQL with schemas: iam, project
- Frontend: Located at `/Users/mayur/git_personal/ui/frontend`

## Testing & Deployment
- Build: `npm run build`
- Deploy: `npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev`
- Lint/Type checking: Always run before committing if commands are known

## API Testing
- Use ID tokens (not access tokens) for Cognito API Gateway
- Test user: buildboard007+555@gmail.com | Mayur@1234

## Testing Guidelines
**NEVER create test files in project root!** Always use organized directories:
- `/testing/api/` - API endpoint tests
- `/testing/auth/` - Authentication & JWT tests
- `/testing/database/` - Database validation (but prefer MCP queries)
- `/testing/utilities/` - Helper scripts & tools

See `/testing/README.md` for full testing process and templates.

## Key Points
1. NEVER create files unless explicitly asked
2. ALWAYS prefer editing existing files
3. NEVER proactively create documentation
4. Keep responses concise and to the point
5. Use MCP for ALL database operations
6. NEVER create test files in root - use `/testing/` directories