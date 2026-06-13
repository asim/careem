# 100-Word Summary

**Smart Code Reviewer** is an AI assistant that gives code a structured
first-pass review *before* it reaches a human reviewer, so people spend their
attention on judgment, not nitpicks. Paste a snippet and it returns a health
summary, a 1–10 score, the **three highest-impact improvements** (each tagged
with category, severity, location, and a concrete fix), and **one genuine
positive note**. My approach: a senior-engineer prompt with anti-quota
guardrails, and — crucially — the output is enforced through a Claude *tool
schema* so it's always valid JSON the UI can render reliably. It's written in
Go (stdlib only) and ships as a web server, a CLI, and a reusable prompt.

<!-- word count: ~100 -->
