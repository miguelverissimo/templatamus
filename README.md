# Templatamus

A simple interactive CLI tool written in Go that generates a new project from a private GitHub repository. It allows you to select a repository, choose a specific tag, branch, or HEAD, and download the corresponding code as a ZIP archive. Optionally, it can initialize a Git repository with an initial commit.

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

## 🧪 Usage Example

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
What's the project name?
> my-app

Downloading...
Unzipping...
Done.

Do you want to init a git repo and initial commit? [Y/n] Y
Committed "initial commit from yourorg/another-repo@v1.0.1"

done!
```

---

## 📄 License

MIT — use this freely in your projects.

