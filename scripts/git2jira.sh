#!/usr/bin/env bash
set -euo pipefail

# git-commits-to-jira.sh (Bash-3 compatible, robust gum + verbose OpenAI)
# Creates Jira tasks from git commits with optional OpenAI grouping, gum review,
# and robust diagnostics for network/payload issues.

# --- Config / Env ---
OPENAI_MODEL="${OPENAI_MODEL:-gpt-4o-mini}"
OPENAI_API_BASE="${OPENAI_API_BASE:-https://api.openai.com/v1}"
OPENAI_API_TIMEOUT="${OPENAI_API_TIMEOUT:-20}"              # seconds
OPENAI_API_CONNECT_TIMEOUT="${OPENAI_API_CONNECT_TIMEOUT:-5}"
OPENAI_MAX_PAYLOAD_KB="${OPENAI_MAX_PAYLOAD_KB:-200}"       # Hard cap for request body (skip AI if larger)
JIRA_ISSUE_TYPE="${JIRA_ISSUE_TYPE:-Task}"
ASSIGNEE_MODE="${ASSIGNEE_MODE:-email}"
VERBOSE="${VERBOSE:-0}"                                     # 1 to be chatty; or pass --verbose
LOG_DIR=".commit2jira"
WORKDIR="$LOG_DIR"
LOG_FILE="$LOG_DIR/run.log"

DRY_RUN=0
USE_AI=1
USE_CONV=0
RANGE_ARG=""
SINCE_ARG=""
UNTIL_ARG=""
BRANCH_ARG=""

mkdir -p "$WORKDIR"
: > "$LOG_FILE"

# --- tiny logger helpers (Bash 3 friendly) ---
ts() { date +"%Y-%m-%d %H:%M:%S"; }
log() { echo "[$(ts)] $*" | tee -a "$LOG_FILE" >&2; }
vlog() { if [[ "$VERBOSE" = "1" ]]; then log "$@"; fi; }
phase() { log "=== $* ==="; }
die() { log "Error: $*"; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"; }

# --- Pre-flight ---
need git
need jq
need curl

GUM_BIN=""
if command -v gum >/dev/null 2>&1; then
  GUM_BIN="$(command -v gum)"
  GUM_VER="$($GUM_BIN --version 2>/dev/null || echo unknown)"
  # Feature detect --title for write (older versions don't have it)
  if "$GUM_BIN" write --help 2>&1 | grep -q -- " --title "; then
    GUM_WRITE_TITLE_SUPPORTED=1
  else
    GUM_WRITE_TITLE_SUPPORTED=0
  fi
else
  GUM_WRITE_TITLE_SUPPORTED=0
fi

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --range) RANGE_ARG="${2:-}"; shift 2;;
    --since) SINCE_ARG="${2:-}"; shift 2;;
    --until) UNTIL_ARG="${2:-}"; shift 2;;
    --branch) BRANCH_ARG="${2:-}"; shift 2;;
    --dry-run) DRY_RUN=1; shift;;
    --no-ai) USE_AI=0; shift;;
    --conv) USE_CONV=1; shift;;
    --verbose) VERBOSE=1; shift;;
    -h|--help)
      cat <<'HLP'
Usage:
  ./scripts/git2jira.sh [--range <git-range>] [--since <date>] [--until <date>] [--branch <main>] [--dry-run] [--no-ai] [--conv] [--verbose]

Examples:
  ./scripts/git2jira.sh --range origin/main..HEAD
  ./scripts/git2jira.sh --since "2025-08-01" --until "2025-09-01"
  ./scripts/git2jira.sh --branch main
  ./scripts/git2jira.sh --range origin/main..HEAD --no-ai --verbose
Env flags:
  VERBOSE=1           extra logs
  OPENAI_DEBUG=1      curl --verbose + payload/response dumps
  OPENAI_MAX_PAYLOAD_KB=200  cap request size before calling OpenAI
HLP
      exit 0
      ;;
    *) die "Unknown arg: $1";;
  esac
done

phase "Environment"
vlog "bash: $(bash --version 2>/dev/null | head -n1 || echo unknown)"
vlog "git:  $(git --version)"
vlog "jq:   $(jq --version)"
vlog "curl: $(curl --version | head -n1)"
vlog "gum:  ${GUM_VER:-not installed}"
vlog "WORKDIR: $WORKDIR  LOG_FILE: $LOG_FILE"
vlog "USE_AI=$USE_AI  OPENAI_MODEL=$OPENAI_MODEL  OPENAI_API_BASE=$OPENAI_API_BASE"
vlog "OPENAI_API_TIMEOUT=$OPENAI_API_TIMEOUT  CONNECT_TIMEOUT=$OPENAI_API_CONNECT_TIMEOUT  MAX_PAYLOAD_KB=$OPENAI_MAX_PAYLOAD_KB"
vlog "JIRA_BASE_URL=${JIRA_BASE_URL:-<unset>}  JIRA_PROJECT_KEY=${JIRA_PROJECT_KEY:-<unset>}"

# --- Validate Jira env ---
: "${JIRA_BASE_URL:?Set JIRA_BASE_URL}"
: "${JIRA_EMAIL:?Set JIRA_EMAIL}"
: "${JIRA_API_TOKEN:?Set JIRA_API_TOKEN}"
: "${JIRA_PROJECT_KEY:?Set JIRA_PROJECT_KEY}"

if [[ "$USE_AI" -eq 1 && -z "${OPENAI_API_KEY:-}" ]]; then
  log "Info: OPENAI_API_KEY not set → falling back to heuristics."
  USE_AI=0
fi

# --- Build range if not provided ---
if [[ -z "$RANGE_ARG" ]]; then
  if [[ -n "${SINCE_ARG}" || -n "${UNTIL_ARG}" ]]; then
    :
  elif [[ -n "${BRANCH_ARG}" ]]; then
    RANGE_ARG="origin/${BRANCH_ARG}..HEAD"
  else
    if git rev-parse --verify origin/main >/dev/null 2>&1; then
      RANGE_ARG="origin/main..HEAD"
    else
      RANGE_ARG="HEAD~50..HEAD"
      log "Info: using last 50 commits (no origin/main). Override with --range."
    fi
  fi
fi

log "▶ Collecting commits for range: ${RANGE_ARG:-<computed args>}"

# --- Prepare git log args as an ARRAY (avoid empty args on Bash 3) ---
GIT_LOG_ARGS=()
if [[ -n "$RANGE_ARG" ]]; then
  # shellcheck disable=SC2206
  TOKENS=( $RANGE_ARG )
  for t in "${TOKENS[@]}"; do
    [[ -n "$t" ]] && GIT_LOG_ARGS+=("$t")
  done
fi
[[ -n "$SINCE_ARG" ]] && GIT_LOG_ARGS+=("--since=${SINCE_ARG}")
[[ -n "$UNTIL_ARG" ]] && GIT_LOG_ARGS+=("--until=${UNTIL_ARG}")

# --- Pre-check range ---
COUNT=$(git rev-list --count "${GIT_LOG_ARGS[@]}" 2>/dev/null || echo 0)
vlog "rev-list count = $COUNT"
if [[ "${COUNT:-0}" -eq 0 ]]; then
  log "ℹ️  No commits found for the given range/constraints. Nothing to do."
  log "Hint: try --range origin/main..HEAD or --since 'YYYY-MM-DD'."
  exit 0
fi

# --- Collect commits safely (ASCII RS/US separators) ---
raw_log=$(git log "${GIT_LOG_ARGS[@]}" --date=iso --pretty=format:'%H%x1f%an%x1f%ae%x1f%ad%x1f%s%x1f%b%x1e')

tmp_commits="$WORKDIR/commits.json"
printf "%s" "$raw_log" \
| jq -Rs '
    split("\u001e")
    | map(select(length>0))
    | map(
        split("\u001f")
        | {
            hash: .[0],
            author: .[1],
            email: .[2],
            date: .[3],
            title: .[4],
            body: (.[5] // "")
          }
      )
  ' > "$tmp_commits"

log "✔ Saved commits to $tmp_commits"
vlog "commits.json size: $(wc -c < "$tmp_commits" | tr -d ' ') bytes"

# --- Attach changed files & stats for each commit (Bash-3 loop) ---
phase "Attach file stats"
jq -r '.[].hash' "$tmp_commits" | while IFS= read -r h; do
  [[ -z "$h" ]] && continue

  files_json="$(
    git diff-tree --no-commit-id --name-status -r "$h" \
      | awk '{printf("{\"status\":\"%s\",\"path\":\"%s\"}\n",$1,$2)}' \
      | jq -s '.'
  )"

  files_changed=$(git show --stat --oneline --pretty="" "$h" | grep -Eo ' [0-9]+ files? changed' | awk '{print $1}' || echo 0)
  ins=$(git show --stat --oneline --pretty="" "$h" | grep -Eo ' [0-9]+ insertions?\(\+\)' | awk '{print $1}' || echo 0)
  del=$(git show --stat --oneline --pretty="" "$h" | grep -Eo ' [0-9]+ deletions?\(-\)' | awk '{print $1}' || echo 0)

  jq --arg h "$h" --argjson files "$files_json" \
     --argjson fch ${files_changed:-0} --argjson ins ${ins:-0} --argjson del ${del:-0} '
    map(if .hash == $h
        then .files = $files
           | .stats = {"files_changed":($fch // 0),"insertions":($ins // 0),"deletions":($del // 0)}
        else .
        end)
  ' "$tmp_commits" > "$tmp_commits.tmp" && mv "$tmp_commits.tmp" "$tmp_commits"
done
vlog "commits.json (post-stats) size: $(wc -c < "$tmp_commits" | tr -d ' ') bytes"

# --- Compact AI input ---
compact_commits="$WORKDIR/commits.compact.json"
jq '
  map({
    hash, date,
    title: (.title // "" | tostring | .[0:160]),
    files: ((.files // []) | map(.path) | unique | .[0:8]),
    stats: (.stats // {files_changed:0,insertions:0,deletions:0})
  })
' "$tmp_commits" > "$compact_commits"

log "▶ Generating tasks with ${USE_AI:-0}==1 ? OpenAI : heuristics"
vlog "compact_commits size: $(wc -c < "$compact_commits" | tr -d ' ') bytes"

draft_tasks="$WORKDIR/tasks.draft.json"

generate_tasks_heuristic () {
  phase "Heuristic task generation"
  jq -r '
    def scope_from_title:
      (capture("(?<type>feat|fix|chore|docs|refactor|perf|test)\\((?<scope>[^)]+)\\):")? // {} );
    map({
      hash, title,
      type: (scope_from_title.type // "change"),
      scope: (scope_from_title.scope // null),
      files: (.files // []),
      stats: (.stats // {files_changed:0,insertions:0,deletions:0})
    })
  ' "$tmp_commits" \
  | jq -s '
      (group_by(.scope // ((.[0].files[0] // "") | split("/")[0] // "misc")) | map({
        scope: (.[0].scope // ((.[0].files[0] // "") | split("/")[0] // "misc")),
        commits: .,
        summary: (
          (.[0].scope // "General") + ": " +
          ([.[].title] | unique | join("; ") | .[0:120])
        ),
        description: (
          "Auto-suggested from commits:\n\n" +
          ( [ .[].title ] | map("- " + .) | join("\n") ) +
          "\n\nFiles touched:\n" +
          ( [ (.[].files[]?) ] | unique | map("- " + .) | join("\n") ) +
          "\n\nStats (sum): " + (
            {
              files:(map(.stats.files_changed // 0)|add),
              insertions:(map(.stats.insertions // 0)|add),
              deletions:(map(.stats.deletions // 0)|add)
            } | tostring
          ) +
          "\n\nAcceptance Criteria:\n- [ ] Changes deployed\n- [ ] Smoke tests pass\n- [ ] Docs updated if needed\n- [ ] Add/Adjust tests"
        ),
        labels: [ (.[0].type // "change"), "commit2jira" ]
      }))
  ' > "$draft_tasks"
  log "✔ Draft tasks (heuristic) → $draft_tasks (size=$(wc -c < "$draft_tasks" | tr -d ' ') bytes)"
}

generate_tasks_ai () {
  phase "OpenAI reachability probe (/v1/models)"
  set +e
  probe_json="$WORKDIR/openai.models.json"
  curl -sS --http1.1 --max-time 8 --connect-timeout 4 \
       -H "Authorization: Bearer ${OPENAI_API_KEY}" \
       "$OPENAI_API_BASE/models" > "$probe_json"
  probe_code=$?
  set -e
  if [[ $probe_code -ne 0 || ! -s "$probe_json" ]]; then
    log "⚠️  Models probe failed (code=$probe_code). Falling back to heuristics."
    generate_tasks_heuristic
    return
  fi
  vlog "Models probe ok (size=$(wc -c < "$probe_json" | tr -d ' ') bytes)"

  phase "Prepare compact request"
  commits_json="$(cat "$compact_commits")"
  prompt=$(cat <<'PP'
You are a senior engineering project manager. Convert the following git commits into a concise set of actionable Jira tasks (3–12 items).
Rules:
- Each task must have: summary (max 120 chars), description (markdown, include rationale & acceptance criteria), and labels.
- Prefer grouping related commits into one task.
- Use imperative mood for summaries (e.g., "Add X", "Fix Y").
- Infer scope/component from file paths.
- If tests are missing for features/bugfixes, add a subtask line in description ("- [ ] Add/Adjust tests").
- Return ONLY compact JSON array with objects: {summary, description, labels[]} — no extra keys or commentary.
PP
)
  prompt_json=$(printf '%s' "$prompt" | jq -Rs .)
  commits_json_str=$(printf '%s' "$commits_json" | jq -Rs .)

  log "• Escaped prompt size: $(printf '%s' "$prompt_json" | wc -c | tr -d ' ') bytes"
  log "• Escaped commits size: $(printf '%s' "$commits_json_str" | wc -c | tr -d ' ') bytes"
  log "• Building OpenAI request body…"

  req_body="$(cat <<EOF
{
  "model": "${OPENAI_MODEL}",
  "messages": [
    {"role":"system","content":"You turn commit logs into Jira tasks."},
    {"role":"user","content": ${prompt_json}},
    {"role":"user","content": ${commits_json_str}}
  ],
  "temperature": 0.2
}
EOF
)"
  bytes=$(printf '%s' "$req_body" | wc -c | tr -d ' ')
  kb=$(( (bytes + 1023) / 1024 ))
  log "• OpenAI request size ~ ${kb} KB"
  if [[ "$kb" -gt "$OPENAI_MAX_PAYLOAD_KB" ]]; then
    log "⚠️  Payload ${kb} KB exceeds limit ${OPENAI_MAX_PAYLOAD_KB} KB → using heuristics."
    generate_tasks_heuristic
    return
  fi
  if [[ "${OPENAI_DEBUG:-0}" = "1" ]]; then
    echo "— OpenAI request head (first 600 chars) —" | tee -a "$LOG_FILE"
    printf '%s' "$req_body" | head -c 600 | sed 's/^/  /' | tee -a "$LOG_FILE"
    echo -e "\n— end —" | tee -a "$LOG_FILE"
  fi

  phase "Call OpenAI chat/completions"
  local_resp="$WORKDIR/openai.response.json"
  set +e
  CURL_ARGS=( -sS --http1.1 --max-time "$OPENAI_API_TIMEOUT" --connect-timeout "$OPENAI_API_CONNECT_TIMEOUT" --fail-with-body )
  [[ "${OPENAI_DEBUG:-0}" = "1" ]] && CURL_ARGS=( --verbose "${CURL_ARGS[@]}" )
  curl "${CURL_ARGS[@]}" \
       -H "Authorization: Bearer ${OPENAI_API_KEY}" \
       -H "Content-Type: application/json" \
       -X POST "${OPENAI_API_BASE}/chat/completions" \
       -d "$req_body" \
       >"$local_resp" 2>>"$LOG_FILE"
  curl_code=$?
  set -e

  vlog "OpenAI HTTP code=$curl_code response size=$(wc -c < "$local_resp" | tr -d ' ') bytes"
  if [[ "${OPENAI_DEBUG:-0}" = "1" ]]; then
    echo "— OpenAI response head (first 600 chars) —" | tee -a "$LOG_FILE"
    (cat "$local_resp" 2>/dev/null | head -c 600 | sed 's/^/  /') | tee -a "$LOG_FILE"
    echo -e "\n— end —" | tee -a "$LOG_FILE"
  fi

  if [[ $curl_code -ne 0 ]]; then
    log "⚠️  OpenAI request failed (curl code=$curl_code) → heuristics."
    generate_tasks_heuristic
    return
  fi

  if ! jq -e . >/dev/null 2>&1 < "$local_resp"; then
    log "⚠️  OpenAI returned non-JSON → heuristics."
    generate_tasks_heuristic
    return
  fi

  content=$(jq -r '.choices[0].message.content // empty' "$local_resp")
  if [[ -z "$content" ]]; then
    log "⚠️  OpenAI returned empty content → heuristics."
    generate_tasks_heuristic
    return
  fi

  if ! printf '%s' "$content" | jq -e . >/dev/null 2>&1; then
    log "⚠️  OpenAI content not JSON array → heuristics."
    generate_tasks_heuristic
    return
  fi

  printf '%s' "$content" | jq 'map({
    summary: .summary,
    description: .description,
    labels: (.labels // [])
  })' > "$draft_tasks"

  log "✔ Draft tasks (OpenAI) → $draft_tasks (size=$(wc -c < "$draft_tasks" | tr -d ' ') bytes)"
}

if [[ "$USE_AI" -eq 1 ]]; then
  log "▶ Generating tasks with OpenAI ($OPENAI_MODEL)"
  generate_tasks_ai
else
  log "▶ Generating tasks with heuristics"
  generate_tasks_heuristic
fi

log "✔ Draft tasks saved to $draft_tasks"

# --- Review (gum) ---
approved_tasks="$WORKDIR/tasks.approved.json"

review_tasks () {
  phase "Review / Approve"
  if [[ -n "$GUM_BIN" ]]; then
    log "▶ Opening gum editor… (Ctrl+S to save, Ctrl+C to abort)"
    tmp_edit="$WORKDIR/edit.tmp.json"
    # Build args compatible with old gum (no --title if unsupported)
    GUM_WRITE_ARGS=( write --width 120 --height 30 )
    if [[ "$GUM_WRITE_TITLE_SUPPORTED" -eq 1 ]]; then
      GUM_WRITE_ARGS+=( --title "Edit & Approve Tasks JSON" )
    fi
    # Write to a file; do NOT capture stderr/stdout into a shell var
    if "$GUM_BIN" "${GUM_WRITE_ARGS[@]}" < "$draft_tasks" > "$tmp_edit"; then
      if [[ -s "$tmp_edit" ]] && jq -e . >/dev/null 2>&1 < "$tmp_edit"; then
        mv "$tmp_edit" "$approved_tasks"
        log "✔ Edits accepted."
      else
        log "⚠️  Edit was empty/invalid JSON → using draft."
        cp "$draft_tasks" "$approved_tasks"
      fi
    else
      log "⚠️  gum write exited non-zero → using draft."
      cp "$draft_tasks" "$approved_tasks"
    fi

    log "▶ Select tasks to create (Space to toggle, Enter to confirm)…"
    choices="$(jq -r 'to_entries[] | "\(.key)\t\(.value.summary)"' "$approved_tasks" || true)"
    if [[ -n "$choices" ]]; then
      selection="$(printf "%s\n" "$choices" | "$GUM_BIN" choose --no-limit --height 15 --header "Pick tasks" | cut -f1 || true)"
      if [[ -n "$selection" ]]; then
        idxs="[$(echo "$selection" | paste -sd, -)]"
        jq --argjson idxs "$idxs" '
          to_entries | map(select(.key as $k | $idxs | index($k))) | map(.value)
        ' "$approved_tasks" > "$approved_tasks.tmp" && mv "$approved_tasks.tmp" "$approved_tasks"
      fi
    fi

    if [[ -n "${JIRA_LABELS:-}" ]]; then
      IFS=',' read -r -a labs <<<"$JIRA_LABELS"
      jq --argjson labs "$(printf '%s\n' "${labs[@]}" | jq -R . | jq -s .)" '
        map(.labels = ((.labels // []) + $labs | unique))
      ' "$approved_tasks" > "$approved_tasks.tmp" && mv "$approved_tasks.tmp" "$approved_tasks"
    fi
  else
    cp "$draft_tasks" "$approved_tasks"
  fi
  log "✔ Approved tasks → $approved_tasks (size=$(wc -c < "$approved_tasks" | tr -d ' ') bytes)"
}

review_tasks

# --- Create Jira issues ---
create_issue () {
  local summary="$1"
  local description="$2"
  local labels_json="$3"

  fields=$(jq -n \
    --arg project "$JIRA_PROJECT_KEY" \
    --arg summary "$summary" \
    --arg desc "$description" \
    --arg issuetype "$JIRA_ISSUE_TYPE" \
    --arg comp "${JIRA_COMPONENT:-}" \
    --arg epic "${JIRA_EPIC_KEY:-}" \
    '{
      project: { key: $project },
      summary: $summary,
      description: $desc,
      issuetype: { name: $issuetype }
    }
    + (if $comp != "" then {components: [{name: $comp}]} else {} end)
    + (if $epic != "" then {"customfield_10014": $epic} else {} end)'
  )

  if [[ -n "${JIRA_ASSIGNEE:-}" ]]; then
    if [[ "$ASSIGNEE_MODE" == "accountId" ]]; then
      fields=$(jq --arg id "$JIRA_ASSIGNEE" '. + {assignee:{accountId:$id}}' <<<"$fields")
    else
      fields=$(jq --arg email "$JIRA_ASSIGNEE" '. + {assignee:{emailAddress:$email}}' <<<"$fields")
    fi
  fi

  if [[ -n "$labels_json" && "$labels_json" != "null" ]]; then
    fields=$(jq --argjson labels "$labels_json" '. + {labels: $labels}' <<<"$fields")
  fi

  payload=$(jq -n --argjson fields "$fields" '{fields:$fields}')

  if [[ $DRY_RUN -eq 1 ]]; then
    log "— DRY RUN — would create issue: $(jq -c . <<<"$payload")"
    return 0
  fi

  resp=$(curl -sS -u "${JIRA_EMAIL}:${JIRA_API_TOKEN}" \
    -H "Content-Type: application/json" \
    -X POST "${JIRA_BASE_URL}/rest/api/3/issue" \
    -d "$payload")

  if echo "$resp" | jq -e '.key' >/dev/null 2>&1; then
    key=$(echo "$resp" | jq -r '.key')
    echo "{\"ok\":true,\"key\":\"$key\",\"summary\":$(jq -R <<<"$summary")}" >> "$WORKDIR/jira.created.jsonl"
    log "✔ Created $key  —  $summary"
  else
    echo "{\"ok\":false,\"error\":$resp}" >> "$WORKDIR/jira.created.jsonl"
    log "✖ Failed creating issue: $resp"
  fi
}

phase "Create Jira issues"
len=$(jq 'length' "$approved_tasks")
i=0
while [[ $i -lt $len ]]; do
  summary=$(jq -r ".[$i].summary" "$approved_tasks")
  description=$(jq -r ".[$i].description" "$approved_tasks")
  labels=$(jq -c ".[$i].labels // []" "$approved_tasks")

  if [[ -z "$summary" || "$summary" == "null" ]]; then
    log "Skipping item $i (missing summary)"
    i=$((i+1))
    continue
  fi

  create_issue "$summary" "$description" "$labels"
  i=$((i+1))
done

phase "Done"
log "Artifacts:"
log "  - $tmp_commits"
log "  - $compact_commits"
log "  - $draft_tasks"
log "  - $approved_tasks"
log "  - $WORKDIR/jira.created.jsonl"
log "  - $LOG_FILE   (full debug log)"

