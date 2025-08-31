[![CI](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/ci.yml/badge.svg)](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/ci.yml) [![Build](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/build.yml/badge.svg)](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/build.yml) [![E2E Test Run](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/e2e-test.yml/badge.svg)](https://github.com/hellej/pr-slack-reminder-action/actions/workflows/e2e-test.yml) [![Coverage Status](https://coveralls.io/repos/github/hellej/pr-slack-reminder-action/badge.svg?branch=main)](https://coveralls.io/github/hellej/pr-slack-reminder-action?branch=main)

# PR Slack Reminder Action

This GitHub Action sends a friendly Slack reminder about open Pull Requests, helping your team stay on top of code reviews and keep the development flow moving.

> ⚠️ **Beta Version Notice**: This action is currently in beta (`v1-beta`). While functional and tested, the API may change before the stable `v1` release planned for October 2025.

## GitHub's Built-in vs This Action

You may not need this action. GitHub provides [built-in scheduled reminders for teams](https://docs.github.com/en/organizations/organizing-members-into-teams/managing-scheduled-reminders-for-your-team), which can work well in many situations.

**When to use GitHub's built-in reminders:**

- Your team structure aligns with GitHub teams
- The CODEOWNERS files of your repositories accurately match your team structure (-> reviews are automatically requested from the right teams)
- You're okay with the 5 repository limit per reminder
- You don't need custom message content or formatting
- You don't need different filtering options for each repository

**What's special about this action:**

- Monitor up to 50 repositories
- Highlight old PRs that need attention (with optional age threshold input)
- Repository specific filters
- No need for official GitHub team setup
- No need for perfect CODEOWNERS files to get reminded about the right PRs
- More customizable Slack message content

## Getting Started

### Prerequisites

- A Slack bot token with permissions to post messages
- [GitHub token](#-github-token-setup) with read access to your repositories

### Basic Usage Examples

#### 1. Simple Setup (Single Repository)

This monitors open PRs in your current repository.

```yaml
name: PR Reminder

on:
  schedule:
    - cron: "0 9 * * MON-FRI" # 9 AM on weekdays

jobs:
  remind:
    runs-on: ubuntu-latest
    steps:
      - uses: hellej/pr-slack-reminder-action@v1-beta
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          slack-bot-token: ${{ secrets.SLACK_BOT_TOKEN }}
          slack-channel-name: "dev-team"
```

#### 2. Multiple Repositories

Monitor several repositories with user mentions and custom messaging.

```yaml
name: PR Reminder

on:
  schedule:
    - cron: "0 9 * * MON-FRI" # 9 AM on weekdays

jobs:
  remind:
    runs-on: ubuntu-latest
    steps:
      - uses: hellej/pr-slack-reminder-action@v1-beta
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          slack-bot-token: ${{ secrets.SLACK_BOT_TOKEN }}
          slack-channel-name: "code-reviews"
          github-repositories: |
            myorg/frontend
            myorg/backend
            myorg/mobile-app
          github-user-slack-user-id-mapping: |
            alice: U1234567890
            kronk: U2345678901
            charlie: U3456789012
          main-list-heading: "We have <pr_count> PRs waiting for review! 👀"
          no-prs-message: "🎉 All caught up! No PRs waiting for review."
          old-pr-threshold-hours: 48
```

#### 3. Advanced Setup with Filtering

Full-featured setup with repository-specific filters and prefixes.

```yaml
name: PR Reminder

on:
  schedule:
    - cron: "0 9 * * MON-FRI"
  workflow_dispatch: # Allow manual triggers

jobs:
  remind:
    runs-on: ubuntu-latest
    steps:
      - uses: hellej/pr-slack-reminder-action@v1-beta
        with:
          github-token: ${{ secrets.MULTI_REPO_GITHUB_TOKEN }}
          slack-bot-token: ${{ secrets.SLACK_BOT_TOKEN }}
          slack-channel-name: "code-reviews"
          github-repositories: |
            myorg/web-app
            myorg/api-service
            myorg/mobile-app
          github-user-slack-user-id-mapping: |
            alice: U1234567890
            kronk: U2345678901
          main-list-heading: "<pr_count> PRs need your attention!"
          no-prs-message: "No PRs pending! Happy coding!"
          old-pr-threshold-hours: 24
          repository-prefixes: |
            web-app: 🌐
            api-service: 📡
            mobile-app: 📞
          filters: |
            {
              "labels-ignore": ["draft", "wip"],
              "authors-ignore": ["dependabot[bot]"]
            }
          repository-filters: |
            api-service: {"labels": ["ready-for-review"], "authors-ignore": ["intern-bot"]}
            mobile-app: {}
```

## Inputs

| Name                                | Required | Description                                                                                                     |
| ----------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------- |
| `github-token`                      | ✅       | GitHub token for repository access<br>Example: `${{ secrets.GITHUB_TOKEN }}`                                    |
| `slack-bot-token`                   | ✅       | Slack bot token for sending messages<br>Example: `${{ secrets.SLACK_BOT_TOKEN }}`                               |
| `slack-channel-name`                | ❌       | Slack channel name (use this OR `slack-channel-id`)<br>Example: `dev-team`                                      |
| `slack-channel-id`                  | ❌       | Slack channel ID (use this OR `slack-channel-name`)<br>Example: `C1234567890`                                   |
| `github-repositories`               | ❌       | Repositories to monitor (defaults to current repo)<br>Example:<br>`owner/repo1`<br>`owner/repo2`                |
| `github-user-slack-user-id-mapping` | ❌       | Map of GitHub usernames to Slack user IDs<br>Example:<br>`alice: U1234567890`<br>`kronk: U2345678901`           |
| `main-list-heading`                 | ❌       | Message heading (`<pr_count>` gets replaced)<br>Example: `There are <pr_count> open PRs 💫`                     |
| `no-prs-message`                    | ❌       | Message when no PRs are found (if not set, no empty message gets sent)<br>Example: `All caught up! 🎉`          |
| `old-pr-threshold-hours`            | ❌       | Duration in hours after which PRs are highlighted as old<br>Example: `48`                                       |
| `repository-prefixes`               | ❌       | Repository specific prefixes to display before PR titles<br>Example:<br>`repo1: 🚀`<br>`repo2: 📦`              |
| `filters`                           | ❌       | Global filters (JSON)<br>Example:<br>`{"authors": ["alice"], "labels-ignore": ["wip"]}`                         |
| `repository-filters`                | ❌       | Repository-specific filters<br>Example:<br>`repo1: {"labels": ["bug"]}`<br>`repo2: {"authors-ignore": ["bot"]}` |

### Filter Options

Both `filters` and `repository-filters` support:

- `authors` - Only include PRs by these users
- `authors-ignore` - Exclude PRs by these users
- `labels` - Only include PRs with these labels
- `labels-ignore` - Exclude PRs with these (overrides the above)

⚠️ **Note**: You cannot use both `authors` and `authors-ignore` in the same filter.

## 🔑 GitHub Token Setup

### Option 1: Default Token (Single Repository)

For monitoring just your current repository, the default `GITHUB_TOKEN` (available automatically) works perfectly:

```yaml
github-token: ${{ secrets.GITHUB_TOKEN }}
```

### Option 2: GitHub App (Recommended for Organizations)

For better security and granular permissions:

1. **Create a GitHub App** in your organization settings
2. **Install the app** on repositories you want to monitor
3. **Use a token generation action** like [actions/create-github-app-token](https://github.com/actions/create-github-app-token)

### Option 3: Personal Access Token (Multiple Repositories)

To monitor multiple repositories, you'll need a Personal Access Token:

1. **Go to GitHub Settings** → Developer settings → Personal access tokens → Fine-grained tokens
2. **Click "Generate new token"** → Select the repositories of interest and at least read access to PRs
3. **Add the token as a repository secret** named `PR_REMINDER_GITHUB_TOKEN`
4. **Use it in your workflow:** `github-token: ${{ secrets.PR_REMINDER_GITHUB_TOKEN }}`

## 💡 Tips

- **Test with `workflow_dispatch`**: Allow manual testing for your workflow
- **Use cron scheduling**: Run reminders at times that work for your team (avoid weekends!)
- **Customize messages**: Make the reminders fit your team's culture
- **Highlight old PRs**: Set a reasonable `old-pr-threshold-hours` to highlight stale PRs (consider weekends too)

## 🤝 Contributing

Found a bug or have a feature request? We'd love your help! Feel free to open an issue.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
