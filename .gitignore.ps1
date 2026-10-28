@'
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out

# Go workspace files
*.mod
*.sum
go.sum

# Database
*.db
*.db-journal

# Environment files
.env
.env.local
.env.production

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Logs
*.log
'@ | Out-File -FilePath .gitignore -Encoding UTF8

# Add .gitignore to git
git add .gitignore
git commit -m "Add .gitignore"