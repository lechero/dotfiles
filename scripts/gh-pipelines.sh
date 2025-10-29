#!/usr/bin/env bash
set -euo pipefail

TMP_VIEW_FILE=""
cleanup() {
  if [[ -n "$TMP_VIEW_FILE" ]]; then
    rm -f "$TMP_VIEW_FILE"
  fi
}
trap cleanup EXIT

choose_run() {
  local menu="${1:-}"
  local choice=""
  local run_id=""
  if ! choice="$(printf '%s\n' "$menu" | gum choose --height 20 --cursor '→ ' --header 'Select a workflow run to view details')"; then
    return $?
  fi
  run_id="${choice##*$'\t'}"
  if [[ -z "${run_id-}" ]]; then
    echo "Unable to determine run id from selection." >&2
    return 1
  fi
  printf '%s' "${run_id-}"
}

show_run_summary() {
  local run_id="${1:-}"
  if [[ -z "$run_id" ]]; then
    echo "Run id is required to view details." >&2
    return 1
  fi

  if [[ -z "${RUN_SUMMARY-}" ]]; then
    TMP_VIEW_FILE="$(mktemp -t gh-run-view.XXXXXX)"
    if ! gum spin --title "Fetching details for run #${run_id}…" -- "${GH_RUN_VIEW[@]}" "$run_id" > "$TMP_VIEW_FILE"; then
      rm -f "$TMP_VIEW_FILE"
      TMP_VIEW_FILE=""
      return 1
    fi
    RUN_SUMMARY="$(cat "$TMP_VIEW_FILE")"
    rm -f "$TMP_VIEW_FILE"
    TMP_VIEW_FILE=""
  fi

  printf '%s\n' "$RUN_SUMMARY"
}

view_run_logs() {
  local run_id="${1:-}"
  if [[ -z "$run_id" ]]; then
    echo "Run id is required to view logs." >&2
    return 1
  fi
  "${GH_RUN_VIEW[@]}" "$run_id" --log | gum pager
}

open_run_in_browser() {
  local run_id="${1:-}"
  if [[ -z "$run_id" ]]; then
    echo "Run id is required to open in browser." >&2
    return 1
  fi
  "${GH_RUN_VIEW[@]}" "$run_id" --web
}

view_job_details() {
  local run_id="${1:-}"
  local jobs_json=""
  local job_menu=""
  local job_choice=""
  local job_id=""
  local job_count=0
  local job_action_choice=""
  local job_action=""

  if [[ -z "$run_id" ]]; then
    echo "Run id is required to view job details." >&2
    return 1
  fi

  if ! jobs_json="$("${GH_RUN_VIEW[@]}" "$run_id" --json jobs)"; then
    echo "Failed to fetch jobs for run #${run_id}." >&2
    return 1
  fi

  job_count="$(jq '.jobs | length' <<<"$jobs_json" 2>/dev/null || echo 0)"
  if [[ "$job_count" -eq 0 ]]; then
    echo "No jobs found for run #${run_id}."
    return 0
  fi

  job_menu="$(
    jq -r '
      .jobs // []
      | sort_by(.startedAt // "")
      | map(
          (if .conclusion == "success" then "✅"
           elif .conclusion == "failure" then "❌"
           elif .status == "in_progress" then "🟡"
           elif .status == "queued" then "⏳"
           else "•" end)
          + "  " + (.name // "Job")
          + "  · " + (.status // "-")
          + (if .conclusion then " (" + .conclusion + ")" else "" end)
          + "  · " + ((.startedAt // "") | sub("\\..*$";""))
          + (if .completedAt then " → " + ((.completedAt // "") | sub("\\..*$";"")) else "" end)
          + "\t" + ((.databaseId // .id) | tostring)
        )
      | .[]
    ' <<<"$jobs_json"
  )"

  if [[ -z "$job_menu" ]]; then
    echo "No jobs available to display for run #${run_id}."
    return 0
  fi

  if ! job_choice="$(printf '%s\n' "$job_menu" | gum choose --height 15 --cursor '→ ' --header "Select a job from run #${run_id} (Esc to skip)")"; then
    return 0
  fi

  job_id="${job_choice##*$'\t'}"
  if [[ -z "$job_id" ]]; then
    echo "Unable to determine job id from selection." >&2
    return 1
  fi

  TMP_VIEW_FILE="$(mktemp -t gh-job-view.XXXXXX)"
  gum spin --title "Fetching job #${job_id} details…" -- "${GH_RUN_VIEW[@]}" "$run_id" --job "$job_id" > "$TMP_VIEW_FILE"
  gum pager <"$TMP_VIEW_FILE" || true
  rm -f "$TMP_VIEW_FILE"
  TMP_VIEW_FILE=""

  local -a job_action_options=(
    $'View job logs\tlogs'
    $'Back\tback'
  )

  while true; do
    if ! job_action_choice="$(
        printf '%s\n' "${job_action_options[@]}" |
        gum choose --height 5 --cursor '→ ' --header "Job #${job_id} actions"
      )"; then
      break
    fi

    job_action="${job_action_choice##*$'\t'}"
    case "$job_action" in
      logs)
        "${GH_RUN_VIEW[@]}" "$run_id" --job "$job_id" --log | gum pager
        printf '\n'
        ;;
      back)
        break
        ;;
      *)
        echo "Unknown job action: $job_action" >&2
        ;;
    esac
  done
}

run_action_menu() {
  local run_id="${1:-}"
  local action_choice=""
  local action=""
  local -a menu_options=(
    $'Inspect jobs\tjobs'
    $'View combined logs\tlogs'
    $'Open run in browser\tbrowser'
    $'Back to run list\tback'
  )

  if [[ -z "$run_id" ]]; then
    echo "Run id is required to select actions." >&2
    return 1
  fi

  while true; do
    if ! show_run_summary "$run_id"; then
      return 1
    fi
    printf '\n'
    if ! action_choice="$(
        printf '%s\n' "${menu_options[@]}" |
        gum choose --height 10 --cursor '→ ' --header "Choose what to view for run #${run_id}"
      )"; then
      break
    fi

    action="${action_choice##*$'\t'}"
    case "$action" in
      jobs)
        view_job_details "$run_id"
        printf '\n'
        ;;
      logs)
        view_run_logs "$run_id"
        printf '\n'
        ;;
      browser)
        open_run_in_browser "$run_id"
        printf '\n'
        ;;
      back)
        break
        ;;
      *)
        echo "Unknown action: $action" >&2
        ;;
    esac
  done
}

# gh-pipelines.sh — Pick a GitHub Actions run with gum and view details/logs.
# Requirements: gh (logged in), jq, gum
# Usage: ./gh-pipelines.sh [-n|--limit N] [-r|--repo owner/repo]
# Examples:
#   ./gh-pipelines.sh
#   ./gh-pipelines.sh -n 30
#   ./gh-pipelines.sh -r topgeschenken/region-tool-backend -n 25

# ---------- deps ----------
for bin in gh jq gum; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "Error: '$bin' is required. Install it (e.g. 'brew install $bin' or 'brew install charmbracelet/tap/gum')." >&2
    exit 1
  fi
done

# ---------- args ----------
LIMIT=20
REPO=""
while [[ $# -gt 0 ]]; do
  case "${1:-}" in
    -n|--limit) LIMIT="${2:-}"; shift 2 ;;
    -r|--repo)  REPO="${2:-}";  shift 2 ;;
    -h|--help)
      sed -n '2,20p' "$0"
      exit 0
      ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

GH_RUN_LIST=(gh run list)
GH_RUN_VIEW=(gh run view)
if [[ -n "${REPO}" ]]; then
  GH_RUN_LIST+=(--repo "$REPO")
  GH_RUN_VIEW+=(--repo "$REPO")
fi

# ---------- fetch runs ----------
JSON_FIELDS="databaseId,displayTitle,workflowName,headBranch,event,status,conclusion,createdAt,url"
if ! RUNS_JSON="$(
  "${GH_RUN_LIST[@]}" --limit "$LIMIT" --json "$JSON_FIELDS"
)"; then
  echo "Failed to list runs. Make sure you're in a repo or pass --repo owner/name." >&2
  exit 1
fi

COUNT="$(jq 'length' <<<"$RUNS_JSON")"
if [[ "$COUNT" -eq 0 ]]; then
  echo "No runs found."
  exit 0
fi

# ---------- build menu ----------
MENU="$(
  jq -r '
    sort_by(.createdAt) | reverse |
    map(
      . as $r |
      (
        (if .status=="completed" and .conclusion=="success" then "✅"
         elif .status=="completed" and .conclusion=="failure" then "❌"
         elif .status=="in_progress" then "🟡"
         elif .status=="queued" then "⏳"
         else "•" end)
        + "  " + (.workflowName // .name // "Workflow")
        + " — " + (.displayTitle // ("Run #" + (.databaseId|tostring)))
        + "  [" + (.headBranch // "-") + "]"
        + "  · " + (.event // "-")
        + "  · " + ((.createdAt // "") | sub("\\..*$";""))
      )
      + "\t" + ($r.databaseId|tostring)
    )
    | .[]
  ' <<<"$RUNS_JSON"
)"

# ---------- choose run ----------
if ! RUN_ID="$(choose_run "$MENU")"; then
  rc=$?
  if [[ $rc -eq 130 ]]; then
    exit 0
  fi
  exit "$rc"
fi
echo "Selected run ID: $RUN_ID"
# ---------- run actions ----------
run_action_menu "$RUN_ID"
