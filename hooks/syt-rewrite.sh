#!/usr/bin/env bash
set -euo pipefail
command -v jq >/dev/null 2>&1 || exit 0
command -v syt >/dev/null 2>&1 || exit 0
INPUT=$(cat)
TOOL=$(echo "$INPUT" | jq -r '.tool_name // ""')
CMD=$(echo "$INPUT" | jq -r '.tool_input.command // ""')
[[ "$TOOL" != "Bash" ]] && exit 0
[[ -z "$CMD" ]] && exit 0
[[ "$CMD" == syt\ * ]] && exit 0
[[ "$CMD" == *$'\n'* ]] && exit 0
REWRITTEN=$(syt rewrite "$CMD" 2>/dev/null) || exit 0
DESC=$(echo "$INPUT" | jq -r '.tool_input.description // ""')
jq -n --arg cmd "$REWRITTEN" --arg desc "$DESC" \
  '{permissionDecision:"allow",updatedInput:{command:$cmd,description:$desc}}'
if [[ "${SYT_HOOK_AUDIT:-0}" == "1" ]]; then
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) REWRITE: $CMD -> $REWRITTEN" \
    >> "${HOME}/.local/share/syt/hook-audit.log"
fi
