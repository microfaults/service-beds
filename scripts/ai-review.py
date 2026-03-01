#!/usr/bin/env python3
"""
AI-powered commit-range review bot for GitLab CI/CD.

Supports Claude (Anthropic) and Gemini (Google) as LLM backends.

Modes:
  Scheduled — auto-resolves FROM_COMMIT using REVIEW_HOURS lookback from HEAD.
  Manual    — caller supplies FROM_COMMIT (and optionally TO_COMMIT).

Output:
  - Saved as a markdown artifact: review-output/review-<to_sha>.md
  - Optionally posted as a commit comment on TO_COMMIT (requires GITLAB_API_TOKEN).

Required CI/CD variables:
  ANTHROPIC_API_KEY - (if using Claude)
  GOOGLE_API_KEY    - (if using Gemini)

Optional CI/CD variables:
  GITLAB_API_TOKEN  - Project access token with 'api' scope; enables commit comments
  LLM_PROVIDER      - "claude" (default) or "gemini"
  LLM_MODEL         - Override model ID
  REVIEW_PROMPT     - Extra review instructions appended to the system prompt
  FROM_COMMIT       - Base commit ref (manual mode); if unset, uses REVIEW_HOURS lookback
  TO_COMMIT         - Head commit ref (default: HEAD)
  REVIEW_HOURS      - Hours to look back for scheduled mode (default: 24)
  MAX_DIFF_CHARS    - Max diff characters sent to LLM (default: 120000)
"""

import json
import os
import subprocess
import sys
import urllib.request
import urllib.error

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

GITLAB_URL = os.environ.get("CI_API_V4_URL", "")
PROJECT_ID = os.environ.get("CI_PROJECT_ID", "")
GITLAB_TOKEN = os.environ.get("GITLAB_API_TOKEN", "")

ANTHROPIC_API_KEY = os.environ.get("ANTHROPIC_API_KEY", "")
GOOGLE_API_KEY = os.environ.get("GOOGLE_API_KEY", "")

LLM_PROVIDER = os.environ.get("LLM_PROVIDER", "claude").lower()
# Model defaults: claude-opus-4-6 for Claude, gemini-3.1-pro for Gemini.
# Override with LLM_MODEL env var if needed.
LLM_MODEL = os.environ.get("LLM_MODEL", "")

REVIEW_PROMPT = os.environ.get("REVIEW_PROMPT", "")
FROM_COMMIT = os.environ.get("FROM_COMMIT", "").strip()
TO_COMMIT = os.environ.get("TO_COMMIT", "HEAD").strip() or "HEAD"
REVIEW_HOURS = int(os.environ.get("REVIEW_HOURS", "24"))
MAX_DIFF_CHARS = int(os.environ.get("MAX_DIFF_CHARS", "120000"))

BOT_MARKER = "<!-- ai-review-bot -->"

REVIEWABLE_EXTENSIONS = {
    ".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".cs", ".rs",
    ".proto", ".yaml", ".yml", ".sh", ".bash", ".sql", ".graphql",
    ".toml", ".mod", ".sum", ".json", ".csproj",
}

OUTPUT_DIR = "review-output"


# ---------------------------------------------------------------------------
# Git helpers
# ---------------------------------------------------------------------------

def git(*args, check=True):
    """Run a git command, return stdout as a string."""
    result = subprocess.run(
        ["git", *args],
        capture_output=True, text=True,
    )
    if check and result.returncode != 0:
        raise RuntimeError(f"git {' '.join(args)} failed:\n{result.stderr.strip()}")
    return result.stdout.strip()


def resolve_head():
    return git("rev-parse", "HEAD")


def find_scheduled_base(hours):
    """
    Return the commit SHA that was HEAD `hours` hours ago.
    Uses the oldest commit in the lookback window's parent as the base.
    Returns None if there are no commits in the window (nothing to review).
    """
    commits_in_window = git(
        "log", f"--since={hours} hours ago", "--format=%H",
    )
    lines = [c for c in commits_in_window.splitlines() if c]
    if not lines:
        return None
    oldest_in_window = lines[-1]
    # Parent of the oldest commit in the window = state before the window
    parent = git("rev-parse", f"{oldest_in_window}^", check=False)
    if not parent:
        # The oldest commit in the window is the very first commit — diff from empty tree
        return git("hash-object", "-t", "tree", "/dev/null")
    return parent


def get_commit_info(sha):
    """Return (short_sha, subject, author) for a commit."""
    fmt = git("log", "-1", "--format=%h|%s|%an", sha)
    parts = fmt.split("|", 2)
    return (parts[0], parts[1] if len(parts) > 1 else "", parts[2] if len(parts) > 2 else "")


def get_commits_in_range(from_sha, to_sha):
    """Return list of (short_sha, subject, author) for commits from_sha..to_sha."""
    log = git("log", "--format=%h|%s|%an", f"{from_sha}..{to_sha}")
    rows = []
    for line in log.splitlines():
        if not line:
            continue
        parts = line.split("|", 2)
        rows.append((parts[0], parts[1] if len(parts) > 1 else "", parts[2] if len(parts) > 2 else ""))
    return rows


def get_diff(from_sha, to_sha):
    """Return filtered unified diff between two commits."""
    raw = git("diff", from_sha, to_sha, "--", *_reviewable_pathspecs())
    if len(raw) > MAX_DIFF_CHARS:
        raw = raw[:MAX_DIFF_CHARS] + "\n\n... [diff truncated — exceeded size limit] ..."
    return raw


def _reviewable_pathspecs():
    """Build git pathspecs for reviewable extensions."""
    specs = [f"*{ext}" for ext in sorted(REVIEWABLE_EXTENSIONS)]
    specs.append("*Dockerfile")
    return specs


# ---------------------------------------------------------------------------
# GitLab API helpers (optional — for posting commit comments)
# ---------------------------------------------------------------------------

def gitlab_api(method, endpoint, body=None):
    url = f"{GITLAB_URL}{endpoint}"
    headers = {"PRIVATE-TOKEN": GITLAB_TOKEN}
    encoded = None
    if body is not None:
        headers["Content-Type"] = "application/json"
        encoded = json.dumps(body).encode()
    req = urllib.request.Request(url, data=encoded, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            raw = resp.read().decode()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode(errors="replace")
        print(f"  GitLab API {exc.code} on {method} {endpoint}: {detail}")
        raise


def get_commit_comments(sha):
    return gitlab_api("GET", f"/projects/{PROJECT_ID}/repository/commits/{sha}/comments")


def post_commit_comment(sha, note):
    return gitlab_api(
        "POST",
        f"/projects/{PROJECT_ID}/repository/commits/{sha}/comments",
        {"note": note},
    )


def already_reviewed(to_sha):
    """True if the bot already posted a comment on this commit."""
    if not GITLAB_TOKEN:
        return False
    try:
        for c in get_commit_comments(to_sha):
            if BOT_MARKER in c.get("note", ""):
                return True
    except Exception:
        pass
    return False


# ---------------------------------------------------------------------------
# LLM calls
# ---------------------------------------------------------------------------

SYSTEM_PROMPT = """\
You are an expert code reviewer. You will receive a unified diff from a \
GitLab commit range together with context about the commits. Ensure code does
not deviate from current style and architecture, especially since most devs
are undergrad CS majors and they deviate towards what works as what's right.

Provide a structured review. For each finding include:
- **File** and line/hunk reference
- **Severity** — critical / high / medium / low / info
- **Category** — security / correctness / performance / style / maintainability
- **Issue** — concise description
- **Suggestion** — how to fix (include a code snippet when helpful)

If the changes look good, say so — do not invent issues. \
End with a brief summary of overall quality and any patterns you noticed.

Format your response as clean GitLab-flavored markdown."""


def _build_user_message(diff_text, ctx, custom_prompt):
    commit_list = "\n".join(
        f"  {sha}  {subject}  ({author})"
        for sha, subject, author in ctx["commits"]
    ) or "  (none)"

    extra = f"\n\n<additional_instructions>\n{custom_prompt}\n</additional_instructions>" if custom_prompt else ""

    return f"""\
<diff>
{diff_text}
</diff>

<commit_range>
From: {ctx['from_sha']} ({ctx['from_subject']})
To:   {ctx['to_sha']} ({ctx['to_subject']})
Branch: {ctx['branch']}

Commits in range:
{commit_list}
</commit_range>{extra}"""


def call_claude(diff_text, ctx, custom_prompt):
    model = LLM_MODEL or "claude-opus-4-6"
    payload = {
        "model": model,
        "max_tokens": 8192,
        "temperature": 0.3,
        "system": SYSTEM_PROMPT,
        "messages": [
            {"role": "user", "content": _build_user_message(diff_text, ctx, custom_prompt)},
        ],
    }
    data = json.dumps(payload).encode()
    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=data,
        headers={
            "x-api-key": ANTHROPIC_API_KEY,
            "anthropic-version": "2023-06-01",
            "content-type": "application/json",
        },
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=600) as resp:
            result = json.loads(resp.read().decode())
        return result["content"][0]["text"]
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"Claude API {exc.code}: {exc.read().decode(errors='replace')}")


def call_gemini(diff_text, ctx, custom_prompt):
    # Verify model ID at https://ai.google.dev/gemini-api/docs/models
    model = LLM_MODEL or "gemini-3.1-pro"
    full_prompt = f"{SYSTEM_PROMPT}\n\n{_build_user_message(diff_text, ctx, custom_prompt)}"
    payload = {
        "contents": [{"parts": [{"text": full_prompt}]}],
        "generationConfig": {"temperature": 0.3, "maxOutputTokens": 8192},
    }
    url = (
        f"https://generativelanguage.googleapis.com/v1beta/models/{model}"
        f":generateContent?key={GOOGLE_API_KEY}"
    )
    data = json.dumps(payload).encode()
    req = urllib.request.Request(
        url, data=data, headers={"content-type": "application/json"}, method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=600) as resp:
            result = json.loads(resp.read().decode())
        return result["candidates"][0]["content"]["parts"][0]["text"]
    except urllib.error.HTTPError as exc:
        raise RuntimeError(f"Gemini API {exc.code}: {exc.read().decode(errors='replace')}")


def call_llm(diff_text, ctx, custom_prompt):
    if LLM_PROVIDER == "gemini":
        if not GOOGLE_API_KEY:
            raise RuntimeError("GOOGLE_API_KEY not set")
        return call_gemini(diff_text, ctx, custom_prompt)
    else:
        if not ANTHROPIC_API_KEY:
            raise RuntimeError("ANTHROPIC_API_KEY not set")
        return call_claude(diff_text, ctx, custom_prompt)


# ---------------------------------------------------------------------------
# Main review flow
# ---------------------------------------------------------------------------

def run_review(from_sha, to_sha, trigger):
    to_sha_full = git("rev-parse", to_sha)
    from_sha_full = git("rev-parse", from_sha)

    _, from_subj, _ = get_commit_info(from_sha_full)
    _, to_subj, _ = get_commit_info(to_sha_full)
    branch = git("rev-parse", "--abbrev-ref", "HEAD", check=False) or "unknown"

    print(f"\n{'=' * 60}")
    print(f"  From : {from_sha_full[:12]}  {from_subj}")
    print(f"  To   : {to_sha_full[:12]}  {to_subj}")
    print(f"  Trigger: {trigger}")
    print(f"{'=' * 60}")

    if already_reviewed(to_sha_full):
        print("  -> Already reviewed this commit. Skipping.")
        return

    diff_text = get_diff(from_sha_full, to_sha_full)
    if not diff_text.strip():
        print("  -> No reviewable file changes in this range. Skipping.")
        return

    commits = get_commits_in_range(from_sha_full, to_sha_full)
    print(f"  -> {len(commits)} commit(s), diff {len(diff_text)} chars")

    ctx = {
        "from_sha": from_sha_full[:12],
        "from_subject": from_subj,
        "to_sha": to_sha_full[:12],
        "to_subject": to_subj,
        "branch": branch,
        "commits": commits,
    }

    print(f"  -> Calling {LLM_PROVIDER} ...")
    review_text = call_llm(diff_text, ctx, REVIEW_PROMPT)

    provider_label = {
        "claude": "Claude (Anthropic)",
        "gemini": "Gemini (Google)",
    }.get(LLM_PROVIDER, LLM_PROVIDER)
    model_used = LLM_MODEL or ("claude-opus-4-6" if LLM_PROVIDER != "gemini" else "gemini-3.1-pro")

    comment = (
        f"{BOT_MARKER}\n"
        f"## AI Code Review — `{from_sha_full[:8]}..{to_sha_full[:8]}`\n\n"
        f"**Model**: {provider_label} (`{model_used}`) | **Trigger**: {trigger}\n\n"
        f"---\n\n"
        f"{review_text}\n\n"
        f"---\n"
        f"<sub>Automated review by AI Review Bot. Findings are advisory.</sub>\n"
    )

    # Save as artifact
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    artifact_path = os.path.join(OUTPUT_DIR, f"review-{to_sha_full[:8]}.md")
    with open(artifact_path, "w") as f:
        f.write(comment)
    print(f"  -> Saved to {artifact_path}")

    # Post as commit comment if token is available
    if GITLAB_TOKEN and GITLAB_URL and PROJECT_ID:
        post_commit_comment(to_sha_full, comment)
        print(f"  -> Posted as commit comment on {to_sha_full[:12]}")
    else:
        print("  -> No GITLAB_API_TOKEN; skipping commit comment (artifact only)")


def main():
    print("AI Review Bot")
    print(f"  Provider : {LLM_PROVIDER}")
    print(f"  Model    : {LLM_MODEL or '(default)'}")

    if FROM_COMMIT:
        # Manual mode — caller provided explicit commits
        print(f"  Mode     : manual ({FROM_COMMIT} .. {TO_COMMIT})")
        print()
        run_review(FROM_COMMIT, TO_COMMIT, trigger="manual")
    else:
        # Scheduled mode — compute base from lookback window
        print(f"  Mode     : scheduled ({REVIEW_HOURS}h lookback)")
        print()
        base = find_scheduled_base(REVIEW_HOURS)
        if base is None:
            print(f"No commits in the last {REVIEW_HOURS}h. Nothing to review.")
            return
        head = resolve_head()
        run_review(base, head, trigger=f"scheduled/{REVIEW_HOURS}h")

    print("\nDone.")


if __name__ == "__main__":
    main()
