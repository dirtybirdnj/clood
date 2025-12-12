# Security Checklist

Answer ONLY "YES" or "NO" for each question. One answer per line.
No explanations. No commentary. Just YES or NO.

---

Code to review:
```
{PASTE CODE HERE}
```

---

Questions:

1. Are there hardcoded passwords, API keys, or secrets?
2. Is user input used directly in SQL queries?
3. Is user input used in shell commands (os.system, subprocess)?
4. Are there eval() or exec() calls?
5. Is sensitive data logged or printed?
6. Are file paths built from user input without validation?
7. Is pickle used on untrusted data?
8. Are weak crypto algorithms used (MD5, SHA1 for security)?
9. Is HTTPS disabled or certificate verification skipped?
10. Are there any TODO or FIXME comments about security?
