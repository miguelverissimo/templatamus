# Templatamus

A simple interactive CLI tool written in Go that generates a new project from a private GitHub repository. It allows you to select a repository, choose a specific tag, branch, or HEAD, and download the corresponding code as a ZIP archive. Optionally, it can initialize a Git repository with an initial commit.

Now with **100% more sync support** to keep your projects up-to-date with their source templates!

## Use at your own risk!
---

## ✨ Features

- Interactive CLI powered by [survey](https://github.com/AlecAivazis/survey)
- Accesses **private GitHub repositories** using a Personal Access Token
- Supports:
  - ✅ Download by **tag**
  - ✅ Download by **branch**
  - ✅ Download from **HEAD (default branch)**
- Unzips and sets up your project in a specified directory
- Optionally runs `git init` and creates the first commit
- **NEW:** Sync with upstream templates:
  - ✅ Track which source repository and commit generated the project
  - ✅ Check for updates from source template
  - ✅ Apply selected commits as separate git commits
  - ✅ Intelligently handle merge conflicts (defers to you!)

---

## 🛠 Installation

```bash
git clone https://github.com/miguelverissimo/templatamus.git
cd templatamus
go mod tidy
go build -o templatamus main.go
```

You can now run the tool using:

```bash
./templatamus
```
or copy it to a directory in your PATH to have it available globally:

```bash
cp templatamus /usr/local/bin/
```

---

## ⚙️ Configuration

Create a JSON config file at `~/.templatamus` with the following structure:

```json
{
  "token": "ghp_yourGitHubToken",
  "repos": [
    "yourorg/repo1",
    "yourorg/repo2"
  ]
}
```

- `token`: Your GitHub **Personal Access Token** with the `repo` scope (see below)
- `repos`: A list of allowed repositories in the format `owner/repo`

---

## 🔑 Generating a GitHub Token

1. Visit: https://github.com/settings/tokens
2. Click **"Generate new token (classic)"**
3. Select the following scope:
   - ✅ `repo` (full control of private repositories)
4. Generate the token and copy it
5. Paste it in your `~/.templatamus` config file

> 💡 If you're part of an organization with SSO, authorize the token after generating it.

---

## 🧪 Usage

### Creating a New Project

```bash
$ ./templatamus
Templatamus 1.0
Choose the repo:
> yourorg/another-repo

You're creating an app from the yourorg/another-repo repository
Do you want to pull head, branch or tag?
> tag

Choose the tag to download:
> v1.0.1

You're creating an app from the yourorg/another-repo repository at tag v1.0.1
Where do you want to create the project? (e.g., myrepo, ../foo, ~/projects/bar)
> my-app

Downloading...
Unzipping...
Done.

Do you want to init a git repo and initial commit? [Y/n] Y
Committed "initial commit from yourorg/another-repo@v1.0.1"

Done!
```

### Syncing with Updates

When run in a directory that was created with Templatamus, it will automatically detect the project and check for updates:

```bash
$ cd my-app
$ ./templatamus
Templatamus 1.0
Found templatamus project at: /home/user/my-app
Checking for updates...
Found 3 new commits.

Available commits:
--------------------------------------------------
1. a1b2c3d4
   Fix typo in README by alice on 2023-04-01

2. e5f6g7h8
   Update dependencies by bob on 2023-04-02

3. i9j0k1l2
   Add new feature by charlie on 2023-04-03

--------------------------------------------------
Select commits to apply (Space to select, Enter to confirm):
> ◉ a1b2c3d4 Fix typo in README
  ◉ e5f6g7h8 Update dependencies
  ◯ i9j0k1l2 Add new feature

Applying commit: a1b2c3d4 - Fix typo in README
Successfully applied commit: a1b2c3d4
Applying commit: e5f6g7h8 - Update dependencies
Successfully applied commit: e5f6g7h8
Sync completed successfully.
Done!
```

### Handling Merge Conflicts

If there are merge conflicts during sync, Templatamus will pause and tell you:

```bash
Applying commit: a1b2c3d4 - Fix typo in README

⚠️  MERGE CONFLICT DETECTED ⚠️
--------------------------------------------------
Commit: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Author: Alice Jones
Date: 2023-04-01 15:04:05
Message: Fix typo in README
--------------------------------------------------
Please resolve the conflicts manually and then continue.

Merge conflicts detected, please resolve manually and run templatamus again
```

When you run Templatamus again after resolving the conflicts:

```bash
$ ./templatamus
Templatamus 1.0
Found templatamus project at: /home/user/my-app
Detected a previous sync with conflicts for commit a1b2c3d4
Have you resolved the conflicts and want to continue? [Y/n] Y
Successfully applied commit a1b2c3d4 with resolved conflicts.
Sync completed successfully.
Done!
```

---

## 📝 Project Metadata

Templatamus stores metadata about your project in a `.templatamus` directory:

```
.templatamus/
├── metadata.json    # Basic project info and applied commits
└── sync.json        # Sync status (only present during conflicts)
```

The `metadata.json` file contains:

```json
{
  "source_repo": "yourorg/template-repo",
  "source_branch": "main",
  "source_commit": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "created_at": "2023-04-01T12:00:00Z",
  "last_synced_at": "2023-04-05T15:30:00Z",
  "applied_commits": [
    "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7"
  ]
}
```

---

## 📄 License

MIT — use this freely in your projects.

