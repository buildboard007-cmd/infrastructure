# Quick Deployment Reference

## 🚀 Deploy to Dev
```bash
cd /Users/mayur/git_personal/infrastructure
npm run build
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

## 🏭 Deploy to Prod  
```bash
cd /Users/mayur/git_personal/infrastructure
npm run build  
cd .. && npx cdk deploy "Infrastructure/Prod/Infrastructure-AppStage" --profile prod
```

## 🔍 Quick Check
```bash
# Verify build works
npm run build

# List stacks
npx cdk list

# Check AWS profiles
aws configure list-profiles
```

## 🆘 Emergency Rollback
```bash
git checkout HEAD~1
cd .. && npx cdk deploy "Infrastructure/Dev/Infrastructure-AppStage" --profile dev
```

---
📖 **Full documentation:** [docs/deployment-guide.md](docs/deployment-guide.md)