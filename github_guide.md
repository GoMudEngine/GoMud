Comprehensive Git Workflow Guide for Forking GoMud
This guide will help you set up a robust Git workflow for customizing the GoMud codebase while maintaining the ability to incorporate upstream updates from the original repository.
Table of Contents
Initial Repository Setup
Branch Strategy
Making Your Custom Changes
Incorporating Upstream Updates
Handling Merge Conflicts
Best Practices
Common Scenarios

Initial Repository Setup
Step 1: Fork the Repository on GitHub
Go to https://github.com/GoMudEngine/GoMud
Click the "Fork" button in the top-right corner
Choose your account as the destination
Wait for GitHub to create your fork
Step 2: Clone Your Fork Locally
# Clone your fork (replace YOUR_USERNAME with your GitHub username)
git clone https://github.com/YOUR_USERNAME/GoMud.git
cd GoMud

Step 3: Add the Original Repository as "upstream"
This is crucial for pulling in updates from the original GoMud repository:
# Add the original repository as a remote called "upstream"
git remote add upstream https://github.com/GoMudEngine/GoMud.git

# Verify your remotes
git remote -v

You should see:
origin    https://github.com/YOUR_USERNAME/GoMud.git (fetch)
origin    https://github.com/YOUR_USERNAME/GoMud.git (push)
upstream  https://github.com/GoMudEngine/GoMud.git (fetch)
upstream  https://github.com/GoMudEngine/GoMud.git (push)

Step 4: Fetch Upstream Branches
# Fetch all branches and tags from upstream
git fetch upstream

# Fetch all branches and tags from your origin
git fetch origin


Branch Strategy
This is the most important part of your workflow. A well-designed branch strategy will make merging updates painless.
Recommended Branch Structure
master/main (tracks upstream/master exactly)
├── development (your main integration branch)
    ├── feature/container-system (individual feature branches)
    ├── feature/randomness-system
    ├── feature/twitch-combat
    └── feature/custom-commands

Step 5: Set Up Your Branch Structure
# Make sure you're on master and it's up to date
git checkout master
git pull upstream master

# Create a development branch - this will be YOUR main branch
git checkout -b development
git push -u origin development

# Create feature branches for each major system change
git checkout -b feature/container-system development
git push -u origin feature/container-system

git checkout development
git checkout -b feature/randomness-system development
git push -u origin feature/randomness-system

git checkout development
git checkout -b feature/twitch-combat development
git push -u origin feature/twitch-combat

Why This Structure?
master: Always matches upstream exactly, never contains your changes. This makes pulling updates trivial.
development: Your main integration branch where all features come together. This is what you'll actually run.
feature branches: Isolated changes for each major system. This makes debugging easier and allows you to disable specific features if they conflict with upstream updates.

Making Your Custom Changes
Step 6: Work on Individual Features
Always work on feature branches, not directly on development:
# Switch to a feature branch
git checkout feature/container-system

# Make your changes to the item/container system
# Edit files, test your changes, etc.

# Commit your changes with descriptive messages
git add internal/items/container.go
git commit -m "feat: make all items containers by default

- Modified Item struct to include container properties
- Updated item creation to initialize container fields
- Added helper methods for container operations"

# Push to your remote repository
git push origin feature/container-system

Step 7: Commit Message Best Practices
Use conventional commit format for clarity:
# Feature additions
git commit -m "feat: implement twitch-style combat system"

# Bug fixes
git commit -m "fix: resolve container nesting issue"

# Refactoring
git commit -m "refactor: reorganize combat calculation logic"

# Documentation
git commit -m "docs: add container system usage guide"

# Configuration changes
git commit -m "chore: update combat config defaults"

Step 8: Merge Features into Development
Once a feature is working:
# Switch to development branch
git checkout development

# Merge the feature
git merge --no-ff feature/container-system -m “<Message>”

# The --no-ff flag creates a merge commit even if it could fast-forward
# This preserves the feature branch history and makes it easier to identify
# which commits belong to which feature

# Push to your remote
git push origin development

The --no-ff flag is important because it:
Creates a clear merge commit showing when a feature was integrated
Makes it easier to revert an entire feature if needed
Preserves the context of your changes

Incorporating Upstream Updates
This is the process you'll follow whenever the GoMud author releases updates.
Step 9: Check for Upstream Updates
# Fetch the latest from upstream (doesn't change your code)
git fetch upstream

# See what's new
git log master..upstream/master --oneline

Step 10: Update Your Master Branch
# Switch to your master branch
git checkout master

# Make sure you haven't accidentally committed to master
git status  # Should say "nothing to commit, working tree clean"

# Pull in upstream changes
git pull upstream master

# Push the updated master to your fork
git push origin master

Your master branch now matches the original repository exactly.
Step 11: Rebase Your Development Branch
Now we'll update your development branch with the new upstream changes:
# Switch to development
git checkout development

# Rebase onto the updated master
# This replays your changes on top of the new upstream code
git rebase master

# If there are conflicts, see "Handling Merge Conflicts" section
# If no conflicts, force push to your remote (rebase rewrites history)
git push --force-with-lease origin development

Important: Use --force-with-lease instead of --force. It's safer because it won't overwrite work if someone else pushed to the branch.
Step 12: Update Feature Branches
For each active feature branch:
# Switch to the feature branch
git checkout feature/container-system

# Rebase onto the updated development branch
git rebase development

# If conflicts occur, resolve them (see next section)
# Push the updated branch
git push --force-with-lease origin feature/container-system

# Repeat for each feature branch
git checkout feature/randomness-system
git rebase development
git push --force-with-lease origin feature/randomness-system

git checkout feature/twitch-combat
git rebase development
git push --force-with-lease origin feature/twitch-combat


Handling Merge Conflicts
Conflicts are inevitable when upstream code changes overlap with your modifications.
Step 13: Understanding Conflict Markers
When a conflict occurs during rebase, Git will stop and show you:
Auto-merging internal/combat/combat.go
CONFLICT (content): Merge conflict in internal/combat/combat.go
error: could not apply abc1234... feat: implement twitch-style combat

The file will contain conflict markers:
<<<<<<< HEAD
// Upstream's version of the code
func CalculateDamage(attacker *Character) int {
    return attacker.Strength * 2
}
=======
// Your version of the code
func CalculateDamage(attacker *Character, defender *Character) int {
    return calculateTwitchDamage(attacker, defender)
}
>>>>>>> feat: implement twitch-style combat

Step 14: Resolving Conflicts
Open the conflicted files in your editor
Decide which code to keep (or combine both):
// Combined resolution
func CalculateDamage(attacker *Character, defender *Character) int {
    // Preserve upstream's base calculation but use twitch system
    baseDamage := attacker.Strength * 2
    return applyTwitchModifiers(baseDamage, attacker, defender)
}

Remove the conflict markers (<<<<<<<, =======, >>>>>>>)
Test your changes to make sure everything still works
Mark the conflict as resolved:
# After editing the file
git add internal/combat/combat.go

# Continue the rebase
git rebase --continue

Step 15: If You Need to Abort
If things get too complicated:
# Abort the rebase and return to the state before you started
git rebase --abort

# You can try again later or use a different strategy

Step 16: Using Interactive Rebase for Complex Situations
If you have many commits and want to clean things up:
# Rebase the last 10 commits interactively
git rebase -i HEAD~10

# You'll see a list of commits:
# pick abc1234 feat: add container system
# pick def5678 fix: container bug
# pick ghi9012 feat: add twitch combat
# pick jkl3456 fix: combat timing

# You can:
# - "pick" to keep a commit as-is
# - "reword" to change the commit message
# - "edit" to modify the commit
# - "squash" to combine with previous commit
# - "drop" to remove the commit

# Example: squash bug fixes into their related features
pick abc1234 feat: add container system
squash def5678 fix: container bug
pick ghi9012 feat: add twitch combat
squash jkl3456 fix: combat timing


Best Practices
Organization Best Practices
Keep Master Clean: Never commit directly to master. It should always mirror upstream exactly.


Small, Focused Commits: Each commit should do one thing. This makes conflicts easier to resolve.

 # Good
git commit -m "feat: add container capacity property"
git commit -m "feat: add container weight limits"

# Bad
git commit -m "various changes to items and combat"


Descriptive Branch Names: Use clear, descriptive names for feature branches.

 # Good
feature/container-system
feature/twitch-combat
fix/inventory-duplication-bug

# Bad
fix-stuff
test
new-feature


Regular Commits: Commit often with meaningful messages. You can always squash commits later.


Document Your Changes: Keep notes about why you made changes, especially when they diverge significantly from upstream.


Testing Best Practices
Test After Every Merge: Always run your MUD and test your features after incorporating upstream changes.


Maintain a Test Checklist: Create a document listing key features to test:

 ## Test Checklist
- [ ] Items can be picked up and dropped
- [ ] All items function as containers
- [ ] Combat damage calculations work
- [ ] Twitch-style combat timing is correct
- [ ] Randomness system produces expected distributions
- [ ] Existing vanilla features still work


Keep a Development Server: Have a test instance where you can try updates before deploying to players.


Backup Best Practices
Tag Stable Versions: Before major updates, tag your working version:

 git tag -a v1.0-stable -m "Stable version before upstream update 2025-01"
git push origin v1.0-stable


Create Backup Branches: Before attempting a complex merge:

 git checkout development
git branch development-backup-2025-01-06
git push origin development-backup-2025-01-06



Common Scenarios
Scenario 1: Upstream Changes a File You Modified
Situation: The upstream changed internal/combat/combat.go and you've heavily modified it for twitch combat.
Solution:
# Update master
git checkout master
git pull upstream master

# Update development with rebase
git checkout development
git rebase master

# Conflict occurs in combat.go
# Open the file, resolve conflicts carefully
# Test to make sure combat still works as expected
git add internal/combat/combat.go
git rebase --continue
git push --force-with-lease origin development

Prevention Strategy: Consider creating wrapper functions instead of modifying core functions:
// Instead of changing original function
func CalculateDamage(attacker *Character) int {
    // your changes
}

// Create a wrapper
func CalculateDamageOriginal(attacker *Character) int {
    return attacker.Strength * 2  // upstream version
}

func CalculateDamage(attacker *Character) int {
    // your twitch combat system
    return calculateTwitchDamage(attacker)
}

Scenario 2: Upstream Adds a Feature You Don't Want
Situation: Upstream adds a new magic system but you want a tech-only MUD.
Solution: You have options:
Keep the feature but disable it: Modify config files to turn it off
Remove the feature: Create a commit that removes it
Let it merge: You can ignore features your players never access
# Option 2: Remove the feature after merging
git checkout development
git rebase master  # Get the upstream changes
# Create a commit that removes/disables the magic system
git commit -m "chore: disable magic system for tech-only theme"

Scenario 3: You Want to Temporarily Disable a Feature
Situation: Your container system has a bug and you want to revert to vanilla items temporarily.
Solution:
# Create a temporary branch without the feature
git checkout master
git checkout -b temp-no-containers
git merge development

# Revert the container feature
git revert <commit-hash-of-container-merge>
git push origin temp-no-containers

# Run your server from this branch until the bug is fixed

Scenario 4: Starting a New Major Feature
Situation: You want to add a crafting system.
Solution:
# Make sure development is up to date first
git checkout development
git pull origin development

# Create the new feature branch
git checkout -b feature/crafting-system

# Make your changes
# ... edit files ...

# Commit and push
git add .
git commit -m "feat: add basic crafting framework"
git push -u origin feature/crafting-system

# When ready, merge into development
git checkout development
git merge --no-ff feature/crafting-system
git push origin development

Scenario 5: Upstream Changes Something You Depend On
Situation: Upstream refactors the item system and your container modifications break.
Solution:
# After the failed rebase
git rebase --abort

# Check what changed upstream
git log master..upstream/master -- internal/items/

# Create a bridge commit on your feature branch
git checkout feature/container-system
# Manually update your code to work with the new upstream structure
git commit -m "refactor: adapt container system to upstream item refactor"

# Now try the rebase again
git checkout development
git rebase master
# Should merge more smoothly now

Scenario 6: You Want to Contribute Back to Upstream
Situation: You've fixed a bug that would benefit the main project.
Solution:
# Create a clean branch from master (no custom changes)
git checkout master
git checkout -b bugfix/inventory-crash

# Cherry-pick just the bug fix commit
git cherry-pick <commit-hash-of-your-fix>

# Make sure it works with vanilla code
# Test thoroughly

# Push to your fork
git push origin bugfix/inventory-crash

# Create a Pull Request on GitHub to upstream


Quick Reference Commands
Daily Workflow
# Start working on a feature
git checkout feature/your-feature
# ... make changes ...
git add .
git commit -m "descriptive message"
git push origin feature/your-feature

# Merge completed feature
git checkout development
git merge --no-ff feature/your-feature
git push origin development

Weekly/Monthly: Check for Updates
git fetch upstream
git checkout master
git pull upstream master
git push origin master
git checkout development
git rebase master
git push --force-with-lease origin development

Emergency: Undo Last Commit (Not Pushed)
git reset --soft HEAD~1  # Keeps changes, removes commit
# or
git reset --hard HEAD~1  # Removes commit AND changes

Emergency: Undo Last Push
git revert HEAD  # Creates a new commit that undoes the last one
git push origin development

See What Changed Upstream
git fetch upstream
git log master..upstream/master
git diff master..upstream/master

Find When Something Broke
git bisect start
git bisect bad  # Current version is broken
git bisect good v1.0-stable  # This version worked
# Git will check out commits for you to test
# After each test:
git bisect good  # or git bisect bad
# When found:
git bisect reset


Helpful Git Configuration
Add these to your Git config for a better experience:
# Show more context in conflicts (3 lines before/after)
git config merge.conflictstyle diff3

# Use a better diff algorithm
git config diff.algorithm histogram

# Auto-prune deleted remote branches
git config fetch.prune true

# Create short aliases
git config alias.st status
git config alias.co checkout
git config alias.br branch
git config alias.last 'log -1 HEAD'
git config alias.unstage 'reset HEAD --'

# Colorful output
git config color.ui true


Troubleshooting
"I accidentally committed to master!"
# Move the commits to a new branch
git branch feature/accidental-changes
git reset --hard origin/master
git checkout feature/accidental-changes

"My rebase created a mess!"
git rebase --abort
# Try merge instead of rebase
git merge master

"I force-pushed and lost my changes!"
# Git keeps old commits for ~30 days in reflog
git reflog
# Find the commit hash before the bad force-push
git reset --hard <commit-hash>

"Conflicts are too complex!"
# Use a visual merge tool
git mergetool

# Or cherry-pick specific commits instead
git rebase --abort
git cherry-pick <specific-commits-you-want>


Conclusion
This workflow gives you:
✅ Clean separation between upstream and your changes
✅ Easy incorporation of upstream updates
✅ Ability to enable/disable features independently
✅ Clear history of what you changed and why
✅ Safety nets for when things go wrong
The key principles:
Master mirrors upstream - always
Development is your main branch - where everything comes together
Features are isolated - one branch per major change
Commit often - small, focused commits
Test after merging - always verify things work
Remember: Git is a tool to help you, not hinder you. If you make a mistake, you can almost always undo it. Don't be afraid to experiment with branches!
Good luck with your MUD development!
