# Clood User Personas

Understanding who builds with local LLM infrastructure.

---

## Overview

Clood serves a spectrum of users—from hardcore developers managing GPU clusters to creative professionals who've never touched a command line. This document identifies key personas to guide feature development, documentation, and marketing.

### The Spectrum

```
TECHNICAL ←────────────────────────────────────────────→ NON-TECHNICAL

Infra        Solo         Homelab      Creative     Small Biz    Privacy-
Engineer     Dev          Builder      Pro          Owner        Conscious
                                                                 Parent
```

---

## Technical Personas

### 1. The Infrastructure Engineer

**Name:** Jordan Chen
**Age:** 34
**Background:** Senior DevOps at a mid-size startup

**Context:**
Jordan manages 50+ services across multiple environments. They've seen the cloud AI bills skyrocket and know there's compute sitting idle on their bare metal servers. They want to add AI capabilities to internal tools without adding another SaaS vendor.

**Goals:**
- Deploy local LLMs across their existing infrastructure
- Integrate AI into CI/CD pipelines (code review, docs generation)
- Keep everything in their private network
- Measure cost per inference vs cloud alternatives

**Pain Points:**
- Cloud AI pricing is unpredictable
- Security team won't approve sending code to external APIs
- Every new AI tool requires its own setup

**What Clood Offers:**
- Multi-host routing across existing servers
- Infrastructure-as-code patterns they already know
- Metrics and benchmarking for capacity planning
- CLI-first approach that integrates with existing workflows

**Quote:**
> "I don't need another dashboard. I need something that fits in my Ansible playbooks."

---

### 2. The Solo Developer

**Name:** Marcus Reid
**Age:** 28
**Background:** Full-stack developer, freelance

**Context:**
Marcus builds web apps for clients and side projects. He uses AI assistants constantly but is tired of copy-pasting between browser tabs. He has a gaming PC with a decent GPU sitting under his desk.

**Goals:**
- Run AI locally while coding
- Avoid API costs that eat into freelance margins
- Experiment with different models for different tasks
- Have AI assistance that understands his codebase

**Pain Points:**
- Claude/GPT subscriptions add up
- Can't share client code with cloud AI
- Context limits force him to re-explain everything
- Wants AI integrated into his terminal workflow

**What Clood Offers:**
- One-command setup on his existing machine
- Model tiers (fast for autocomplete, deep for architecture)
- Project context loading for codebase awareness
- The Saga for maintaining conversation continuity

**Quote:**
> "I want copilot-level assistance without sending my clients' code to someone's server."

---

### 3. The Homelab Enthusiast

**Name:** Sam Okonkwo
**Age:** 41
**Background:** Network engineer, hobby server builder

**Context:**
Sam has a rack in the basement: a couple of old workstations, a mini PC cluster, and a NAS with 100TB. They run everything from Plex to Home Assistant. Adding AI to the homelab is the next frontier.

**Goals:**
- Use spare compute for something useful
- Build a "server garden" of AI capabilities
- Self-host AI rather than rely on external services
- Learn about LLMs without cloud abstractions

**Pain Points:**
- Most AI tools assume cloud deployment
- Documentation written for Docker experts, not hardware tinkerers
- Unclear which models run on which hardware
- Wants to see their servers actually working

**What Clood Offers:**
- Multi-host architecture designed for heterogeneous hardware
- Benchmarking tools to find optimal model-to-hardware mapping
- Visual "server garden" showing the whole setup
- Hardware profiling that speaks their language

**Quote:**
> "My servers shouldn't just be space heaters. I want to see them think."

---

## Semi-Technical Personas

### 4. The Creative Professional

**Name:** Elena Vasquez
**Age:** 36
**Background:** UX designer at an agency

**Context:**
Elena uses AI daily—for copy, wireframe ideas, user research synthesis. She's comfortable with basic terminal commands from design tool installations but wouldn't call herself technical. She has a MacBook Pro with an M2 chip.

**Goals:**
- AI assistance for design work (copy, research, ideation)
- Keep client data private
- Simple setup that doesn't require sysadmin skills
- Something that "just works" when she needs it

**Pain Points:**
- NDA-restricted client work can't go to cloud AI
- Technical setup guides assume too much knowledge
- Doesn't want to learn DevOps
- Worried about breaking things

**What Clood Offers:**
- One-command install (Homebrew)
- Sensible defaults that work on Mac hardware
- `clood chat` for conversation without configuration
- Documentation written for humans, not robots

**Quote:**
> "I just want to ask it questions about my project without worrying about where that data goes."

---

### 5. The Small Business Owner

**Name:** David Park
**Age:** 52
**Background:** Owns a local accounting firm, 8 employees

**Context:**
David has heard about AI transforming businesses. He's seen competitors use chatbots and automation. His firm handles sensitive financial data that absolutely cannot leave their network. His "IT person" is his nephew who visits on weekends.

**Goals:**
- AI assistance for document processing
- Automate repetitive client communications
- Keep all data on-premises for compliance
- Something his staff can use without training

**Pain Points:**
- Cloud AI is a non-starter for financial data
- Enterprise AI solutions cost $50k+/year
- Doesn't have technical staff to manage complex systems
- Needs guaranteed data privacy for audits

**What Clood Offers:**
- Runs on a single office server (Mac mini fits the budget)
- Air-gapped operation with no external calls
- Simple web interface option (future)
- Documentation on compliance and data isolation

**Quote:**
> "I can't tell my clients their tax returns went through ChatGPT. But I need this technology to compete."

---

### 6. The Privacy-Conscious Professional

**Name:** Rachel Torres
**Age:** 45
**Background:** Family law attorney

**Context:**
Rachel needs to draft documents, summarize cases, and research precedents. AI would save hours daily. But attorney-client privilege means she cannot use cloud AI under any circumstances. She's not technical but understands the stakes.

**Goals:**
- AI for legal document drafting and research
- Absolute guarantee that data never leaves her office
- Something her paralegal can also use
- Audit trail for bar compliance

**Pain Points:**
- Ethical rules prohibit cloud AI for client matters
- Legal-specific AI tools are expensive and cloud-based
- Can't risk even accidental data exposure
- Needs to explain her AI use to clients

**What Clood Offers:**
- Air-gapped deployment with network isolation
- Local storage with no telemetry
- Audit logging (future feature)
- Plain-English explanation of data handling

**Quote:**
> "My clients trust me with their most sensitive information. I will not betray that trust to save time."

---

## Non-Technical Personas

### 7. The Curious Parent

**Name:** Michelle and James Thompson
**Ages:** 44 and 46
**Background:** High school teacher and civil engineer

**Context:**
Their kids are using AI for homework, social media, and who knows what else. They want to understand this technology—really understand it—not just use someone else's black box. They have an old desktop that could maybe run something.

**Goals:**
- Understand how AI actually works (demystify it)
- Create a safe, local AI for family use
- Teach kids about technology ownership
- No corporate surveillance of their family

**Pain Points:**
- "AI" is scary and opaque
- Don't know where to start
- Technical guides are intimidating
- Worried about kids' data being harvested

**What Clood Offers:**
- Educational documentation about how LLMs work
- Family-friendly setup guide (Sunday project level)
- Complete local control—no accounts, no tracking
- Conversational interface their kids can use too

**Quote:**
> "We don't want to be afraid of AI. We want to understand it, together."

---

### 8. The Maker/Hobbyist

**Name:** Tom Brennan
**Age:** 58
**Background:** Retired electrician, Arduino enthusiast

**Context:**
Tom builds things—robots, sensors, automated garden systems. He's heard about local LLMs and imagines voice-controlled assistants, smart tools, maybe even a robot that can reason. He has a Raspberry Pi collection and doesn't mind tinkering.

**Goals:**
- Add "intelligence" to physical projects
- Voice commands for workshop automation
- Something that runs on small hardware
- Community of fellow makers to share ideas

**Pain Points:**
- AI tutorials assume cloud APIs
- Doesn't want ongoing subscription costs
- Edge deployment documentation is sparse
- Needs help bridging hardware and software

**What Clood Offers:**
- Documentation on running small models on ARM
- Maker project templates (voice assistant, etc.)
- Offline-first operation for remote/workshop use
- Discord community for sharing builds

**Quote:**
> "If my tomato plants can tell me when they're thirsty, why can't they have a conversation about it?"

---

### 9. The Educator

**Name:** Dr. Patricia Okonkwo
**Age:** 55
**Background:** CS professor at a community college

**Context:**
Patricia teaches intro programming and wants to add AI concepts to her curriculum. She can't rely on cloud AI—students don't all have accounts, and she needs reproducible environments. Her department has a few lab machines available.

**Goals:**
- Demonstrate AI concepts without cloud dependencies
- Reproducible classroom environments
- Students can experiment without API costs
- Teach about AI responsibility and data privacy

**Pain Points:**
- Cloud AI requires accounts and credit cards
- Rate limits hit when 30 students try simultaneously
- Can't guarantee consistent behavior for grading
- Wants to teach how AI works, not just how to use it

**What Clood Offers:**
- Lab deployment guide for shared infrastructure
- Educational materials about model architecture
- Consistent local inference for reproducible demos
- Discussion materials on AI ethics and privacy

**Quote:**
> "I want my students to understand AI, not just consume it."

---

## Marketing Implications

### For Technical Users

**Messaging:** Power and control
- "Your infrastructure, your models, your rules"
- "CLI-native, GitOps-ready, no vendor lock-in"
- Focus on: benchmarks, performance, integration

**Channels:**
- Hacker News, Reddit (r/selfhosted, r/homelab)
- Dev.to, personal blogs
- Conference talks (DevOps, MLOps)

### For Semi-Technical Users

**Messaging:** Simplicity with capability
- "Local AI that respects your work"
- "Private AI for professionals"
- Focus on: privacy, ease of use, specific use cases

**Channels:**
- LinkedIn, professional newsletters
- Industry-specific publications
- Word of mouth in professional communities

### For Non-Technical Users

**Messaging:** Understanding and ownership
- "Own your AI"
- "AI you can trust, because you control it"
- Focus on: education, safety, family values, independence

**Channels:**
- YouTube tutorials
- Local maker spaces
- Parent tech blogs
- Library workshops

---

## Feature Prioritization by Persona

| Feature | Infra | Solo Dev | Homelab | Creative | Small Biz | Privacy Pro | Parent | Maker | Educator |
|---------|-------|----------|---------|----------|-----------|-------------|--------|-------|----------|
| Multi-host | High | Low | High | Low | Low | Low | Low | Med | Med |
| CLI tools | High | High | High | Med | Low | Low | Low | Med | Med |
| Web UI | Low | Med | Med | High | High | High | High | Low | High |
| Benchmarks | High | Med | High | Low | Low | Low | Low | Med | Med |
| Privacy docs | Med | Med | Low | Med | High | High | Med | Low | Med |
| Tutorials | Low | Med | Med | High | High | High | High | High | High |
| The Saga | Med | High | Med | High | Med | Med | Med | Low | Med |

---

## The Server Garden Metaphor

This metaphor works across personas:

**Technical:** "It's a cluster, but think of it as a garden you tend"
**Semi-Technical:** "Your computers working together like a garden growing food"
**Non-Technical:** "A garden of helpers, each good at different things"

Visual concepts:
- Servers as plants (different species = different models)
- The driver (your laptop) as the gardener
- Health metrics as plant health
- Pruning/optimizing as tending

---

*This document should evolve as we learn from actual users.*
