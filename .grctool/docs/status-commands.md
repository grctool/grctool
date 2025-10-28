# Status Commands Reference

> Usage guide for status commands and filtering options

---

**Generated**: 2025-10-28 10:33:19 EDT
**GRCTool Version**: dev
**Documentation Version**: dev

---

## Overview

The `grctool status` command provides visibility into evidence state across all tasks. Use it to understand what evidence needs to be collected, what's been generated, and what's ready for submission.

---

## Basic Commands

### Dashboard View
```bash
grctool status
```

Shows summary statistics:
- Evidence state counts (no_evidence, generated, submitted, etc.)
- Automation level breakdown
- Recent activity
- Next steps recommendations

### Task Details
```bash
grctool status task ET-0047
```

Shows detailed information for a specific task:
- Current state
- Evidence files in root directory (count, size, timestamps)
- Files in `.submitted/` directory (if any)
- Files in `archive/` directory (if any)
- Generation metadata
- Submission status
- Applicable tools

### Force Rescan
```bash
grctool status scan
```

Forces a fresh scan of evidence directories, bypassing cache.

---

## Filtering Options

### Filter by State
```bash
# Tasks with no evidence
grctool status --filter state=no_evidence

# Generated but not submitted
grctool status --filter state=generated

# Submitted tasks
grctool status --filter state=submitted
```

### Filter by Automation Level
```bash
# Fully automated tasks
grctool status --filter automation=fully_automated

# Manual tasks
grctool status --filter automation=manual_only
```

### Combined Filters
```bash
# Fully automated tasks with no evidence (highest priority)
grctool status \
  --filter state=no_evidence \
  --filter automation=fully_automated
```

---

## Understanding Output

### State Indicators
| State | Meaning | Location | Action |
|-------|---------|----------|--------|
| 🔴 no_evidence | No files | Empty window | Generate evidence |
| 🟡 generated | Files exist | Root directory | Evaluate (optional) |
| 🟢 evaluated | Quality scored | Root + `.validation/` | Review or Submit |
| 📤 submitted | Sent to Tugboat | `.submitted/` directory | Wait for review |
| ✅ accepted | Approved | `.submitted/` + `archive/` | Done |
| ❌ rejected | Needs rework | `.submitted/` | Move back to root, regenerate |

### Automation Levels
| Level | Meaning |
|-------|----------|
| 🤖 fully_automated | Can be fully automated with tools |
| ⚙️  partially_automated | Some automation possible |
| 👤 manual_only | Requires manual collection |
| ❓ unknown | Automation level not determined |

---

## Common Use Cases

### "What needs to be done?"
```bash
grctool status --filter state=no_evidence
```

### "What can Claude automate?"
```bash
grctool status --filter automation=fully_automated --filter state=no_evidence
```

### "What's ready to submit?"
```bash
# Evidence that's been evaluated
grctool status --filter state=evaluated

# Or evidence that's in root directory (generated)
grctool status --filter state=generated
```

### "What's been submitted?"
```bash
grctool status --filter state=submitted
```

### "Show me everything for ET-0047"
```bash
grctool status task ET-0047
```

### "Check files in root directory (working area)"
```bash
# List working files
ls evidence/ET-0047_*/2025-Q4/*.md

# Check if files exist in root
ls -lh evidence/ET-0047_*/2025-Q4/
```

### "Check files in .submitted/ directory"
```bash
# List submitted files
ls evidence/ET-0047_*/2025-Q4/.submitted/

# Check submission metadata
cat evidence/ET-0047_*/2025-Q4/.submitted/.submission/submission.yaml
```

### "Check synced evidence from Tugboat"
```bash
# List archived files
ls evidence/ET-0047_*/2025-Q4/archive/

# Check Tugboat metadata
cat evidence/ET-0047_*/2025-Q4/archive/.submission/submission.yaml
```

---

**Next Steps**: Use status output to prioritize evidence collection with `evidence-workflow.md` guidance.
